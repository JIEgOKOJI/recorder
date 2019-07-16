package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"./av/pubsub"
	//	"./format/fmp4"
	"github.com/Jeffail/gabs"
	"github.com/nareix/joy4/av"

	//	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/format"
)

type Client struct {
	id                     string
	stopRecord             chan []byte
	cntrl                  *Controller
	archivePath            string
	livePath               string
	exitsCounter           int
	archivePathWithoutMins string
	que                    *pubsub.Queue
	streams                []av.CodecData
	isTransmuxing          bool
	isPrem                 bool
}

func init() {
	format.RegisterAll()
}

func makeRequest(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Read body: %v", err)
	}

	return string(data), nil
}
func DownloadChunkFile(filepath string, url string, pl string, chunk string, duration float64, client *Client) error {

	// Create the file
	//fmt.Println(filepath)
	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		err := WritePlaylist(pl, chunk, duration, client)
		if err != nil {
			fmt.Println(err)
		}
		out, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer out.Close()

		// Get the data
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err

		}
		go appenQue(client, filepath)
		//		go OnFlyTransmux(filepath, client.archivePath, client.id)
	} else {
		//fmt.Println("exist")
		//client.exitsCounter++
	}
	return nil
}
func appenQue(client *Client, tsFilepath string) {
	//file, _ := avutil.Open(tsFilepath)
	//avutil.CopyFile(client.que, file)
	//client.streams, _ = file.Streams()
	//fmt.Println(client.isTransmuxing)
	if client.isTransmuxing != true {
		fmt.Println("Starting on fly transmux")
		if !strings.Contains(client.id, "_720") && !strings.Contains(client.id, "_480") && !strings.Contains(client.id, "_240") {
			go OnFlyTransmux(client.archivePath, client.id, client, client.livePath+client.id+"_vod.m3u8")
		}
		client.isTransmuxing = true
	}
	//	file.Close()
}
func OnFlyTransmux(mp4path string, id string, client *Client, hlspath string) {
	existsAndMake(mp4path)
	mp4path = mp4path + id + ".mp4"
	ffmpeg, err := exec.Command("/usr/bin/ffmpeg", "-y", "-i", hlspath, "-c", "copy", "-movflags", "+skip_trailer", "-f", "mp4", mp4path).Output()
	if err != nil {
		fmt.Println(string(ffmpeg))
		fmt.Println(fmt.Sprint(err))
		return
	}
	fmt.Println(string(ffmpeg))
	/*
		fmt.Println(mp4path)
		mp4path = mp4path + id + ".mp4"
		outfile, _ := os.Create(mp4path)
		dst := fmp4.NewMuxer(outfile)
		dst.SetPath("/tank/")
		dst.SetMaxFrames(350)
		dst.WriteHeader(client.streams, true)
		err := avutil.CopyPackets(dst, client.que.Latest())
		if err != nil {
			log.Println(err)
		}*/
	log.Println("EndMux")

}

/*func OnFlyTransmux(tsFilepath string, mp4path string, id string) {
	existsAndMake(mp4path)
	mp4path = mp4path + id + ".mp4"
	mp4path_init := mp4path + id + "_init.mp4"
	file, _ := avutil.Open(tsFilepath)
	// Create the fileformat
	_, err := os.Stat(mp4path)
	if os.IsNotExist(err) {
		outfile, err := os.Create(mp4path)
		defer outfile.Close()
		if err != nil {
			fmt.Println(err)
		}
		dst := fmp4.NewMuxer(outfile)
		dst.SetPath(mp4path)
		dst.SetMaxFrames(60)
		fileStreams, _ := file.Streams()
		dst.WriteHeader(fileStreams, true)
		err = avutil.CopyPackets(dst, file)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		outfile, err := os.OpenFile(mp4path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println(err)
		}
		defer outfile.Close()
		dst := fmp4.NewMuxer(outfile)
		dst.SetPath(mp4path_init)
		dst.SetMaxFrames(60)
		fileStreams, _ := file.Streams()
		dst.WriteHeader(fileStreams, false)
		err = avutil.CopyPackets(dst, file)
		if err != nil {
			fmt.Println(err)
		}
	}
}*/
func DownloadFile(filepath string, url string) error {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
func WritePlaylist(filepath string, data string, duration float64, client *Client) error {

	// Create the file
	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		out, err := os.Create(filepath)
		if err != nil {
			existsAndMake(client.archivePathWithoutMins)
			existsAndMake(client.livePath)
			return err
		}
		defer out.Close()
		header := "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-PLAYLIST-TYPE:LIVE\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-TARGETDURATION:2\n#EXTINF:" + fmt.Sprintf("%f", duration) + ",\n"
		// Write the body to file
		_, err = out.WriteString(header + data)
		if err != nil {
			return err
		}
		//fmt.Println("Created")
	} else {
		f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println(err, "here")
		}

		defer f.Close()

		if _, err = f.WriteString("\n#EXTINF:" + fmt.Sprintf("%f", duration) + ",\n" + data); err != nil {
			fmt.Println(err, "orhere")
		}
		//fmt.Println("append")
	}

	return nil
}
func fetchStream(streamName string, path string, client *Client) {
	if client.isPrem {
		streamName = streamName + "_prem"
	}
	playlist, err := makeRequest("http://hls.goodgame.ru/hls/" + streamName + ".m3u8")
	if err != nil {
		fmt.Println(err)
		if err.Error() == "Status error: 404" {
			if client.exitsCounter > 5 {
				client.cntrl.unregister <- client
			} else {
				client.exitsCounter++
			}
		}
	}
	playlistString := strings.Split(playlist, "\n")
	var dur float64
	for _, line := range playlistString {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "#EXTINF:"):
			sepIndex := strings.Index(line, ",")
			duration := line[8:sepIndex]
			durationFloat, _ := strconv.ParseFloat(duration, 64)
			//fmt.Println(durationFloat)
			dur = durationFloat
		case !strings.HasPrefix(line, "#"):
			//fmt.Println(line)
			fetch2(line, dur, streamName, path, client)
		}
	}
}
func fetch2(chunk string, Duration float64, NAME string, path string, client *Client) {
	DownloadChunkFile(path+string(chunk), "http://hls.goodgame.ru/hls/"+chunk, path+client.id+"_vod.m3u8", chunk, Duration, client)
}
func endPlaylist(filepath string) {
	fmt.Println("playlist path : ", filepath)
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println(err, "here")
	}

	defer f.Close()

	if _, err = f.WriteString("\n#EXT-X-ENDLIST"); err != nil {
		fmt.Println(err, "orhere")
	}
}
func hlsToMp4(filepath string, id string) {
	hlspath := filepath + id + "_vod.m3u8"
	mp4path := filepath + id + ".mp4"
	fmt.Println("hls : ", hlspath, "mp4 : ", mp4path)
	ffmpeg, err := exec.Command("/usr/bin/ffmpeg", "-y", "-i", hlspath, "-live_start_index", "0", "-movflags", "+faststart", "-c:a", "copy", "-c:v", "copy", "-f", "mp4", mp4path).Output()
	if err != nil {
		fmt.Println(fmt.Sprint(err))
		return
	}
	fmt.Println(string(ffmpeg))
	deletePlaylist, _ := exec.Command("find", filepath, "-name", "*.m3u8", "-type", "f", "-delete").Output()
	deleteChunks, _ := exec.Command("find", filepath, "-name", "*.ts", "-type", "f", "-delete").Output()
	fmt.Println("delete playlist: ", string(deletePlaylist), " delete chunks:", string(deleteChunks))
	mp4box, _ := exec.Command("mp4box", "-inter", "5000", mp4path, "-tmp", filepath).Output()
	fmt.Println("Mp4Box faststart: ", string(mp4box))
	return
}
func (c *Client) handlerRead() {
	for {
		select {
		case _, ok := <-c.stopRecord:
			if ok {
				log.Println("StopRecord1")
				endPlaylist(c.livePath + c.id + "_vod.m3u8")
				if !strings.Contains(c.id, "_720") && !strings.Contains(c.id, "_480") {
					subj := "mp4"
					jsonObj := gabs.New()
					jsonObj.Set(c.archivePath, "path")
					jsonObj.Set(c.id, "name")
					c.que.Close()
					c.cntrl.nc.Publish(subj, jsonObj.Bytes())
					c.cntrl.nc.Flush()
					if err := c.cntrl.nc.LastError(); err != nil {
						fmt.Println(err)
					} else {
						fmt.Printf("Published [%s] : '%s'\n", subj, jsonObj.Bytes())
					}
				}
				go func() {
					err := os.Rename(c.livePath, "/hot/vod/"+c.id+"/temp-"+time.Now().Format("20060102150405")+"/")
					if err != nil {
						log.Println(err)
					}
					err = os.RemoveAll("/hot/vod/" + c.id + "/temp-" + time.Now().Format("20060102150405") + "/")
					if err != nil {
						log.Println(err)
					}
				}()

				return
			}
		default:
			/*if c.exitsCounter > 10 {
				go checkForLive(c)
			}*/
			//log.Println(c.exitsCounter)
			fetchStream(c.id, c.livePath, c)
			time.Sleep(1 * time.Second)
			//log.Println("recording ...")

		}
	}

}
func checkForLive(c *Client) {
	//log.Println("check for live")
	result, err := makeRequest("https://goodgame.ru/api/player?src=" + c.id)
	if err != nil {
		log.Println("Error while parse api")
		return
	}
	jsonParsed, _ := gabs.ParseJSON([]byte(result))
	channel_status := jsonParsed.Path("channel_status").Data().(string)
	if channel_status == "offline" {
		log.Println("it is offline")
		c.cntrl.unregister <- c
	} else {
		c.exitsCounter = 0
	}

}
func existsAndMake(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		//fmt.Println("Directory Exist")
		return true
	}
	if os.IsNotExist(err) {
		fmt.Println("Making Dir: " + path)
		merr := os.MkdirAll(path, os.ModePerm)
		if merr != nil {
			fmt.Println("Error making Dir: ", merr)
			return false
		}
		return false
	}
	return true
}

func getArchivePath(id string) (string, string) {
	currentTime := time.Now()
	timeStampString := currentTime.Format("2006-01-02 15:04:05")
	layOut := "2006-01-02 15:04:05"
	timeStamp, err := time.Parse(layOut, timeStampString)
	if err != nil {
		fmt.Println(err)
	}
	hr, min, sec := timeStamp.Clock()
	path := "/tank/vod/" + id + "/" + strconv.Itoa(currentTime.Year()) + "/" + currentTime.Month().String() + "/" + strconv.Itoa(currentTime.Day()) + "/" + strconv.Itoa(hr) + "-" + strconv.Itoa(min) + "-" + strconv.Itoa(sec) + "/"
	pathWithoutMins := "/tank/vod/" + id + "/" + strconv.Itoa(currentTime.Year()) + "/" + currentTime.Month().String() + "/" + strconv.Itoa(currentTime.Day()) + "/"
	return path, pathWithoutMins
}
func Recorder(controller *Controller, id string) {
	log.Println("StartRecord")
	var ArchivePath string
	var ArchivePathWithoutMins string
	var LivePath string
	var prem bool = false
	if strings.Contains(id, "_prem") {
		t := strings.Replace(id, "_prem", "", -1)
		id = t
		prem = true
		ArchivePath, ArchivePathWithoutMins = getArchivePath(t)
		LivePath = "/hot/vod/" + t + "/live/"
	} else {
		ArchivePath, ArchivePathWithoutMins = getArchivePath(id)
		LivePath = "/hot/vod/" + id + "/live/"
	}
	que := pubsub.NewQueue()
	que.SetMaxBufCount(30)
	client := &Client{id: id, stopRecord: make(chan []byte, 256), cntrl: controller, archivePath: ArchivePath, livePath: LivePath, exitsCounter: 0, archivePathWithoutMins: ArchivePathWithoutMins, que: que, isTransmuxing: false, isPrem: prem}
	client.cntrl.register <- client
	existsAndMake(client.archivePathWithoutMins)
	existsAndMake(client.livePath)
	client.handlerRead()
}

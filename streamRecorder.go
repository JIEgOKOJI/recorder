package main

import (
	"Recorder/joy4/format/rtmp"
	"Recorder/joy4/format/ts"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	//	"Recorder/joy4/av"

	"Recorder/joy4/av/avutil"
	"Recorder/joy4/av/pubsub"
	"Recorder/joy4/format"
	"Recorder/joy4/format/mp4"

	"strings"
	"time"

	"github.com/Jeffail/gabs"
)

func init() {
	format.RegisterAll()
}

type Client struct {
	id           string
	stopRecord   chan []byte
	startRecord  chan []byte
	cntrl        *Controller
	archivePath  string
	livePath     string
	exitsCounter int
	mut          *sync.Mutex
	chann        *map[string]int
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

func (c *Client) handlerRead() {
	fmt.Println(c.id, "RECORD START")
	go pull(c)
	for {
		select {
		case _, ok := <-c.stopRecord:
			if ok {
				log.Println("StopRecord")

				go cleanUp(c)
				return
			}
		case _, ok := <-c.startRecord:
			if ok {
				log.Println("StartRecord")

				return
			}
		default:
			time.Sleep(1 * time.Second)
			//log.Println("recording ...")

		}
	}
	fmt.Println(c.id, "RECORD STOP ADN EXIT")

}
func SendNats(c *Client) {
	if !strings.Contains(c.id, "_720") && !strings.Contains(c.id, "_480") {
		subj := "mp4"
		jsonObj := gabs.New()
		jsonObj.Set(c.archivePath, "path")
		jsonObj.Set(c.id, "name")
		c.cntrl.nc.Publish(subj, jsonObj.Bytes())
		c.cntrl.nc.Flush()
		if err := c.cntrl.nc.LastError(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Published [%s] : '%s'\n", subj, jsonObj.Bytes())
		}
	}
}
func cleanUp(c *Client) {
	//	tempdir := "/hot/vod/" + c.id + "/temp-" + time.Now().Format("20060102150405") + "/"
	err := os.Rename(c.livePath, "/hot/vod/"+c.id+"/temp-"+time.Now().Format("20060102150405")+"/")
	if err != nil {
		log.Println(err)
	}
	files, err := filepath.Glob("/hot/vod/" + c.id + "/temp-*")
	if err != nil {
		log.Println(err)
	}
	//err = os.RemoveAll(tempdir)
	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			log.Println(err)
		}
	}
	/*if err != nil {
		log.Println(err)
	}*/
}
func pull(c *Client) {
	fmt.Println("Pulling and Finding")
	var at int = 0
	//var codecdata []av.CodecData
attemp:
	go cleanUp(c)
	if !strings.Contains(c.id, "_720") && !strings.Contains(c.id, "_480") {
		existsAndMake(c.archivePath)
	}
	existsAndMake(c.livePath)
	src := "rtmp://origin-7.goodgame.ru:1938/" + c.id
	conn, err := rtmp.Dial(src)
	if err != nil {
		fmt.Println(err)
		fmt.Println(c.id)
	}
	_, err = conn.Streams()
	if err == io.EOF {
		fmt.Println("StreamNotFound")
		if at < 5 {
			at += 1
			goto attemp
		} else {
			go cleanUp(c)
			fmt.Println("ExitPull not found")
			return
		}
	}
	c.mut.Lock()
	(*c.chann)[c.id] = 1
	c.mut.Unlock()
	timeStart := int32(time.Now().Unix())
	que := pubsub.NewQueue()
	que.SetMaxGopCount(3)
	if !strings.Contains(c.id, "_720") && !strings.Contains(c.id, "_480") {
		go func() {
			time.Sleep(3 * time.Second)
			/*	defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered in f", r)
				}
			}()*/
			PATH := c.archivePath
			outfile, err := os.Create(PATH + c.id + ".mp4")
			fmt.Println(outfile, err)
			mp4_f := mp4.NewMuxer(outfile)
			cursor := que.Oldest()
			//			codecdata, _ = conn.Streams()
			//mp4_f.WriteHeader(codecdata)
			err = avutil.CopyFile(mp4_f, cursor)
			if err != nil {
				fmt.Println("MP4 COPY FILE ERROR ", err, " name ", c.id)
			}
			/*err = mp4_f.WriteTrailer()
			if err != nil {
				fmt.Println("MP4 WriteTrailer Error ", err)
			}*/
			outfile.Close()
			log.Println("stop")
		}()
	}
	go func() {
		time.Sleep(3 * time.Second)
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered in f", r)
			}
		}()
		log.Println("start")
		PATH := c.livePath
		outfile, err := os.Create(PATH + "/" + c.id + "-0.ts")
		fmt.Println(PATH + "/" + c.id + "-0.ts")
		tsmux := ts.NewMuxer(c.id, PATH, outfile)
		tsmux.WritePlaylist()
		cursor := que.Latest()
		err = avutil.CopyFile(tsmux, cursor)
		if err != nil {
			fmt.Println(err)
		}
		log.Println("stop")
		outfile.Close()
	}()
	strms, _ := conn.Streams()
	fmt.Println("CODEC DATA ", len(strms))
	err = avutil.CopyFile(que, conn)

	if err != nil {
		if err == io.EOF {
			fmt.Println("STREAM ENDED ", at)
			/*if at < 5 {
				at += 1
				goto attemp
			}*/

		}
	}
	time.Sleep(3 * time.Second)
	que.Close()

	//c.cntrl.unregister <- c
	timeStop := int32(time.Now().Unix())
	go cleanUp(c)
	if timeStop-timeStart >= 120 {
		go SendNats(c)
	} else {
		fmt.Println("Video too short don't publish")
	}

	c.mut.Lock()
	delete((*c.chann), c.id)
	c.mut.Unlock()
	fmt.Println("ExitPull")
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
func Recorder(controller *Controller, id string, l *sync.Mutex, cha *map[string]int) {
	log.Println("StartRecord")
	archivePath, _ := getArchivePath(id)
	livePath := "/hot/vod/" + id + "/live"
	client := Client{id: id, stopRecord: make(chan []byte, 256), cntrl: controller, archivePath: archivePath, livePath: livePath, exitsCounter: 0, mut: l, chann: cha}
	//client.cntrl.register <- &client
	client.handlerRead()
}

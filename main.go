// Recorder project main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	//	"github.com/nareix/joy4/format/mp4"
	//"github.com/nareix/joy4/av/pktque"
	"./av/pubsub"
	"./format/fmp4"
	"github.com/nareix/joy4/format"
	"github.com/nats-io/go-nats"
)

var subj = "trns"
var showTime = flag.Bool("t", false, "Display timestamps")

func init() {
	format.RegisterAll()
}
func transmux(que *pubsub.Queue, streams []av.CodecData) {
	outfile, _ := os.Create("/tank/all2.mp4")
	dst := fmp4.NewMuxer(outfile)
	dst.SetPath("/tank/")
	dst.SetMaxFrames(5)
	dst.WriteHeader(streams, true)
	err := avutil.CopyPackets(dst, que.Oldest())
	if err != nil {
		log.Println(err)
	}
	log.Println("EndMux")
	que.Close()
}
func main() {
	/*log.Println("Hello World!")
	que := pubsub.NewQueue()
	que.SetMaxBufCount(155)
	file, _ := avutil.Open("/tank/vod1168-5532.ts")
	streams, _ := file.Streams()
	avutil.CopyFile(que, file)
	go transmux(que, streams)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5533.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5534.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5535.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5536.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5537.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5538.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5539.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5540.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5541.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5542.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5542.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5543.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5544.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5545.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5546.ts")
	avutil.CopyFile(que, file)
	file.Close()
	file, _ = avutil.Open("/tank/vod1168-5547.ts")
	avutil.CopyFile(que, file)
	file.Close()
	que.Close()

	/*OnFlyTransmux("/tank/vod1168-5532.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5533.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5534.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5535.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5536.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5537.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5538.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5539.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5540.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5541.ts", "/tank/", "test")
	OnFlyTransmux("/tank/vod1168-5542.ts", "/tank/", "test")*/
	nc, err := nats.Connect("nats://guest:guest@origin-7.goodgame.ru:4242")

	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}
	cntrlr := newController()
	go cntrlr.run()
	nc.Subscribe(subj, func(msg *nats.Msg) {
		go printMsg(msg, cntrlr)
		//time.Sleep(2 * time.Second)
	})
	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s]\n", subj)
	if *showTime {
		log.SetFlags(log.LstdFlags)
	}
	//go Recorder(cntrlr, "10603")
	runtime.Goexit()
}
func printMsg(m *nats.Msg, cntrlr *Controller) {
	log.Println(string(m.Data))
	jsonParsed, _ := gabs.ParseJSON(m.Data)
	action := jsonParsed.Path("action").Data().(string)
	name := jsonParsed.Path("name").Data().(string)
	log.Println(action, "   ", name)
	if name == "6147" || name == "54105" || name == "48195" {
		return
	}
	Premium, err := checkForPrem(name)
	if err != nil {
		log.Println(err)
		return
	}
	if Premium != true {
		return
	}
	if action == "start" {
		for k, c := range cntrlr.records {
			if k == name {
				cntrlr.unregister <- c
				return
			}

		}
		//if name == "10603" {
		time.Sleep(3 * time.Second)
		go Recorder(cntrlr, name+"_prem")
		genMaster(name)
		time.Sleep(3 * time.Second)
		go Recorder(cntrlr, name+"_720")
		go Recorder(cntrlr, name+"_480")
		//}

	} else {

	}
}
func genMaster(streamname string) {
	prem_bitrate := "8000"
	prem_resol := "1920x1080"
	high_bitrate := "2000"
	pathtomanifest := "/tank/vod"
	manifest := "#EXTM3U\r\n" + "#EXT-X-VERSION:3\r\n" + "#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=" + prem_bitrate + "000,RESOLUTION=" + prem_resol + "\r\n" + streamname + "/live/" + streamname + "_vod.m3u8\r\n" + "#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=" + high_bitrate + "000,RESOLUTION=1280x720\r\n" + streamname + "_720" + "/live/" + streamname + "_720_vod.m3u8\r\n" + "#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=1320000,RESOLUTION=800x450\r\n" + streamname + "_480" + "/live/" + streamname + "_480_vod.m3u8\r\n"
	/*err := os.Remove(pathtomanifest + "/" + streamname + "_master.m3u8")
	if err != nil {
		fmt.Println(err)
	}*/
	f, err := os.Create(pathtomanifest + "/" + streamname + "_master.m3u8")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	f.Write([]byte(manifest))
}
func checkForPrem(name string) (bool, error) {
	var hidden float64
	//log.Println("check for live")
	result, err := makeRequest("https://goodgame.ru/api/player?src=" + name)
	if err != nil {
		return false, err
	}
	jsonParsed, _ := gabs.ParseJSON([]byte(result))
	children, _ := jsonParsed.ChildrenMap()
	for key, value := range children {
		switch value.Data().(type) {
		case nil:
			log.Println("interface api data is nil")
		default:
			switch key {
			case "hidden":
				hidden = jsonParsed.Path("hidden").Data().(float64)
			case "channel_premium":
				switch jsonParsed.Path("channel_premium").Data().(type) {
				case bool:
					channel_status := jsonParsed.Path("channel_premium").Data().(bool)
					return channel_status, nil
				case string:
					return false, nil
				}
			}
		}

	}

	if hidden == 1 {
		return false, errors.New("channel is hidden")
	}

	return false, errors.New("can't find type in json answer")

}

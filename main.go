// Recorder project main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"Recorder/joy4/av/pubsub"

	"github.com/Jeffail/gabs"
	"github.com/nats-io/go-nats"
)

var subj = "trns"
var showTime = flag.Bool("t", false, "Display timestamps")

type Channel struct {
	que *pubsub.Queue
}

var Channels map[string]int

func main() {
	l := &sync.Mutex{}
	Channels = make(map[string]int)
	log.Println("Hello World!")
	nc, err := nats.Connect("nats://guest:guest@origin-7.goodgame.ru:4242")

	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}
	cntrlr := newController()
	go cntrlr.run()
	nc.Subscribe(subj, func(msg *nats.Msg) {
		printMsg(msg, cntrlr, l)
		time.Sleep(2 * time.Second)
	})
	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s]\n", subj)
	if *showTime {
		log.SetFlags(log.LstdFlags)
	}
	runtime.Goexit()
}
func printMsg(m *nats.Msg, cntrlr *Controller, l *sync.Mutex) {
	//log.Println(string(m.Data))
	jsonParsed, _ := gabs.ParseJSON(m.Data)
	action := jsonParsed.Path("action").Data().(string)
	name := jsonParsed.Path("name").Data().(string)
	//log.Println(action, "   ", name)
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
		genMaster(name)
		l.Lock()
		time.Sleep(10 * time.Second)
		if _, ok := Channels[name]; ok {

			//check if already punlishing
		} else {
			go Recorder(cntrlr, name, l, &Channels)
		}
		l.Unlock()
		l.Lock()
		if _, ok := Channels[name+"_720"]; ok {

			//check if already punlishing
		} else {
			go Recorder(cntrlr, name+"_720", l, &Channels)
		}
		l.Unlock()
		l.Lock()
		if _, ok := Channels[name+"_480"]; ok {

			//check if already punlishing
		} else {
			go Recorder(cntrlr, name+"_480", l, &Channels)
		}
		l.Unlock()
	} else {

	}
}
func genMaster(streamname string) {
	prem_bitrate := "8000"
	prem_resol := "1920x1080"
	high_bitrate := "2000"
	pathtomanifest := "/tank/vod"
	manifest := "#EXTM3U\r\n" + "#EXT-X-VERSION:3\r\n" + "#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=" + prem_bitrate + "000,RESOLUTION=" + prem_resol + "\r\n" + streamname + "/live/" + streamname + ".m3u8\r\n" + "#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=" + high_bitrate + "000,RESOLUTION=1280x720\r\n" + streamname + "_720" + "/live/" + streamname + "_720.m3u8\r\n" + "#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=1320000,RESOLUTION=800x450\r\n" + streamname + "_480" + "/live/" + streamname + "_480.m3u8\r\n"
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
	//	var hidden float64
	//log.Println("check for live")
	result, err := makeRequest("https://goodgame.ru/api/4/recorder/" + name)
	if err != nil {
		return false, err
	}
	//log.Println(result)
	if result == "true" {
		log.Println("Api said record then we record it ", result)
		return true, nil
	}
	if name == "10603" {
		return true, nil
	}

	return false, errors.New("can't find type in json answer " + name)

}

// Recorder project main.go
package main

import (
	"errors"
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/nats-io/go-nats"
)

var subj = "trns"
var showTime = flag.Bool("t", false, "Display timestamps")

func main() {
	log.Println("Hello World!")
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
			}

		}
		time.Sleep(3 * time.Second)
		go Recorder(cntrlr, name)

	} else {

	}
}
func checkForPrem(name string) (bool, error) {
	//log.Println("check for live")
	result, err := makeRequest("https://goodgame.ru/api/player?src=" + name)
	if err != nil {
		return false, err
	}
	jsonParsed, _ := gabs.ParseJSON([]byte(result))
	switch jsonParsed.Path("channel_premium").Data().(type) {
	case bool:
		channel_status := jsonParsed.Path("channel_premium").Data().(bool)
		return channel_status, nil
	case string:
		return false, nil
	}
	return false, errors.New("can't find type in json answer")

}

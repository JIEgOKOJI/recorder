// Recorder project main.go
package main

import (
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
		printMsg(msg, cntrlr)
		time.Sleep(2 * time.Second)
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
	jsonParsed, _ := gabs.ParseJSON(m.Data)
	action := jsonParsed.Path("action").Data().(string)
	name := jsonParsed.Path("name").Data().(string)
	log.Println(action, "   ", name)
	if name == "6147" {
		return
	}
	/*for k, _ := range cntrlr.records {
		if k == name {
			return
		}
		if k == "6147" {
			return
		}
	}*/
	go Recorder(cntrlr, name)
}

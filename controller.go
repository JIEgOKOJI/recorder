package main

import (
	"log"
)

type Controller struct {
	records    map[string]*Client
	register   chan *Client
	unregister chan *Client
}

func newController() *Controller {
	return &Controller{
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		records:    make(map[string]*Client),
	}
}
func (C *Controller) run() {
	for {
		select {
		case client := <-C.register:
			client.cntrl.records[client.id] = client
			log.Println(client.id)
		case client := <-C.unregister:
			client.stopRecord <- []byte("1")
			delete(client.cntrl.records, client.id)
			close(client.stopRecord)
			log.Println(client.id)
		}
	}
}

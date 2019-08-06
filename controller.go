package main

import (
	"log"
	"sync"

	"github.com/nats-io/go-nats"
)

type Controller struct {
	records    map[string]*Client
	register   chan *Client
	unregister chan *Client
	nc         *nats.Conn
	mu         *sync.Mutex
}

func newController() *Controller {
	connection, _ := nats.Connect("nats://guest:guest@origin-7.goodgame.ru:4242")
	return &Controller{
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		records:    make(map[string]*Client),
		mu:         &sync.Mutex{},
		nc:         connection,
	}
}
func (C *Controller) run() {
	for {
		select {
		case client := <-C.register:
			C.mu.Lock()
			client.cntrl.records[client.id] = client
			C.mu.Unlock()
			client.startRecord <- []byte("1")
			log.Println("START RECORD CNTRL")
			log.Println(client.id)
		case client := <-C.unregister:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Println("Recovered in f", r)
					}
				}()
				log.Println("STOP RECORD CNTRL")
				client.stopRecord <- []byte("1")
			}()
			C.mu.Lock()
			delete(client.cntrl.records, client.id)
			C.mu.Unlock()
			close(client.stopRecord)
			log.Println(client.id)
		}
	}
}

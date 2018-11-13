package main

import (
	//	"fmt"
	"log"
	"sync"

	"github.com/nats-io/go-nats"
)

type Controller struct {
	records    map[string]*Client
	register   chan *Client
	unregister chan *Client
	mu         *sync.Mutex
	nc         *nats.Conn
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
			log.Println(client.id)
		case client := <-C.unregister:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Println("Recovered in f", r)
					}
				}()
				client.stopRecord <- []byte("1")
			}()

			C.mu.Lock()
			delete(client.cntrl.records, client.id)
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Println("Recovered in f", r)
					}
				}()
				close(client.stopRecord)
			}()
			C.mu.Unlock()
			log.Println(client.id)
		}
	}
}

package main

import (
	"log"
	"sync"
)

type Controller struct {
	records    map[string]*Client
	register   chan *Client
	unregister chan *Client
	mu         *sync.Mutex
}

func newController() *Controller {
	return &Controller{
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		records:    make(map[string]*Client),
		mu:         &sync.Mutex{},
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

// Package messenger provides broadcasting mechanism
package messenger

import "fmt"

// Messenger object
type Messenger struct {
	get       chan chan interface{}
	del       chan chan interface{}
	broadcast chan interface{}
	pool      map[chan interface{}]struct{}
	reset     chan struct{}
	kill      chan struct{}
}

// New creates new Messenger
func New() *Messenger {
	m := &Messenger{}
	m.get = make(chan chan interface{})
	m.del = make(chan chan interface{})
	m.broadcast = make(chan interface{})
	m.pool = make(map[chan interface{}]struct{})
	m.reset = make(chan struct{})
	m.kill = make(chan struct{})
	go m.monitor()
	return m
}

// Main loop where all the action happens
func (m *Messenger) monitor() {
	tmp := make(chan interface{})
	for {
		select {
		case m.get <- tmp:
			m.pool[tmp] = struct{}{}
			tmp = make(chan interface{})
		case del := <-m.del:
			if _, ok := m.pool[del]; ok {
				close(del)
				delete(m.pool, del)
			}
		case <-m.reset:
			for k := range m.pool {
				close(k)
				delete(m.pool, k)
			}
		case <-m.kill:
			for k := range m.pool {
				close(k)
				delete(m.pool, k)
			}
			close(m.get)
			return
		case msg := <-m.broadcast:
			for k := range m.pool {
				k <- msg
			}
		}
	}
}

// Reset removes and closes all clients.
func (m *Messenger) Reset() {
	m.reset <- struct{}{}
}

// Kill removes and closes all clients and stops the reading and writing goroutine.
func (m *Messenger) Kill() {
	m.kill <- struct{}{}
}

// Sub subscribes a new client for reading broadcasts.
// Clients should be always listening or broadcasting will block.
// Clients should check whether the channel is closed or not.
func (m *Messenger) Sub() (client chan interface{}, err error) {
	sub, ok := <-m.get
	if !ok {
		return nil, fmt.Errorf("can't sub, messenger killed")
	}
	return sub, nil
}

// Unsub unsubscribes a client.
func (m *Messenger) Unsub(client chan interface{}) {
	for {
		select {
		case <-client:
		case m.del <- client:
			return
		}
	}
}

// Broadcast broadcasts a message to all current clients.
// If a client is not listening this will block.
func (m *Messenger) Broadcast(msg interface{}) {
	m.broadcast <- msg
}

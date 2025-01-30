package events

import (
	"sync"
)

type Receiver struct {
	listeners []chan interface{}
	mu        sync.Mutex
}

func New() *Receiver {
	return &Receiver{
		listeners: make([]chan interface{}, 0),
	}
}

func (er *Receiver) Listen() <-chan interface{} {
	er.mu.Lock()
	defer er.mu.Unlock()
	ch := make(chan interface{})
	er.listeners = append(er.listeners, ch)
	return ch
}

func (er *Receiver) Send(event interface{}) {
	er.mu.Lock()
	defer er.mu.Unlock()
	for _, ch := range er.listeners {
		ch <- event
	}
}

func (er *Receiver) Close() {
	er.mu.Lock()
	defer er.mu.Unlock()
	for _, ch := range er.listeners {
		close(ch)
	}
	er.listeners = make([]chan interface{}, 0)
}

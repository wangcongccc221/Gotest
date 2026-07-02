package events

import (
	"errors"
	"sync"
)

var ErrPublisherNotConfigured = errors.New("events publisher not configured")

type Publisher interface {
	PublishJSON(topic string, jsonText string) error
}

type PublisherFunc func(topic string, jsonText string) error

func (fn PublisherFunc) PublishJSON(topic string, jsonText string) error {
	return fn(topic, jsonText)
}

var publisherState = struct {
	sync.RWMutex
	publisher Publisher
}{}

func SetPublisher(publisher Publisher) {
	publisherState.Lock()
	publisherState.publisher = publisher
	publisherState.Unlock()
}

func SetPublisherFunc(fn func(topic string, jsonText string) error) {
	if fn == nil {
		SetPublisher(nil)
		return
	}
	SetPublisher(PublisherFunc(fn))
}

func PublishJSON(topic string, jsonText string) error {
	publisherState.RLock()
	publisher := publisherState.publisher
	publisherState.RUnlock()
	if publisher == nil {
		return ErrPublisherNotConfigured
	}
	return publisher.PublishJSON(topic, jsonText)
}

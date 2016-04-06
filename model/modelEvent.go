package model

import "fmt"

type ModelEvent struct {
	eventType string
	model     interface{}
}

func (me *ModelEvent) String() string {
	return fmt.Sprintf("%s on  %s", me.eventType, me.model)
}

func NewModelEvent(eventType string, model interface{}) *ModelEvent {
	return &ModelEvent{eventType, model}
}

func FromInterfaceChannel(fromChannel chan interface{}) chan *ModelEvent {
	result := make(chan *ModelEvent)
	go func() {
		for {
			event := <-fromChannel
			if evt, ok := event.(*ModelEvent); ok {
				result <- evt
			} else {
				panic(event)
			}

		}
	}()
	return result

}

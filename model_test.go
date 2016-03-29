package main

import (
	"errors"
	. "github.com/arkenio/goarken/model"
	"github.com/arkenio/goarken/storage"
	"github.com/coreos/go-etcd/etcd"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

type MockServiceDriver struct {
	calls  map[string]int
	events *Broadcaster
}

func NewMockServiceDriver() *MockServiceDriver {
	return &MockServiceDriver{
		calls:  make(map[string]int),
		events: NewBroadcaster(),
	}
}

func (sd *MockServiceDriver) Create(s *Service, startOnCreate bool) (interface{}, error) {
	sd.calls["create"] = sd.calls["create"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return s, nil
}

func (sd *MockServiceDriver) Start(s *Service) (interface{}, error) {
	sd.calls["start"] = sd.calls["start"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return s, nil
}

func (sd *MockServiceDriver) Stop(s *Service) (interface{}, error) {
	sd.calls["stop"] = sd.calls["stop"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return s, nil

}
func (sd *MockServiceDriver) Destroy(s *Service) error {
	sd.calls["destroy"] = sd.calls["destroy"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return nil
}

func (sd *MockServiceDriver) Listen() chan *ModelEvent {
	result := make(chan *ModelEvent)
	FromInterfaceChannel(sd.events.Listen(), result)
	return result
}

func (w *MockServiceDriver) StopDriver() {

}

func Test_EtcdWatcher(t *testing.T) {
	if os.Getenv("IT_Test") != "" {
		IT_EtcdWatcher(t)
	}
}

func IT_EtcdWatcher(t *testing.T) {
	var model *Model
	client := etcd.NewClient([]string{})
	sd := NewMockServiceDriver()
	Convey("Given a model", t, func() {
		client.Delete("/domains", true)
		client.Delete("/services", true)

		pd := storage.NewWatcher(client, "/services", "/domains")
		model = NewArkenModel(sd, pd)

		Convey("When i create a service", func() {

			service := &Service{}
			service.Init()

			service.Name = "testService"

			model.Create(service, false)

			Convey("Then the service should be available in all services", func() {
				//Has to wait for 2 update events
				me, _ := waitModelEvent(pd)
				me, _ = waitModelEvent(pd)
				So(me, ShouldNotBeNil)

				So(len(model.Services), ShouldEqual, 1)
				sc := model.Services["testService"]
				So(sc, ShouldNotBeNil)
			})

		})

	})
}

func waitModelEvent(pd PersistenceDriver) (*ModelEvent, error) {
	channel := pd.Listen()
	ticker := time.NewTicker(time.Duration(5) * time.Second)

	select {
	case <-ticker.C:
		return nil, errors.New("Timeout when waiting for ModelEvent")
	case event := <-channel:
		return event, nil
	}

}

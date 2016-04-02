package main

import (
	. "github.com/arkenio/goarken/model"
	"github.com/arkenio/goarken/storage"
	"github.com/coreos/go-etcd/etcd"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
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
	return &RancherInfoType{EnvironmentId: "rancherId"}, nil
}

func (sd *MockServiceDriver) Start(s *Service) (interface{}, error) {
	sd.calls["start"] = sd.calls["start"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return &RancherInfoType{EnvironmentId: "rancherId"}, nil
}

func (sd *MockServiceDriver) Stop(s *Service) (interface{}, error) {
	sd.calls["stop"] = sd.calls["stop"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return &RancherInfoType{EnvironmentId: "rancherId"}, nil

}
func (sd *MockServiceDriver) Destroy(s *Service) error {
	sd.calls["destroy"] = sd.calls["destroy"] + 1
	sd.events.Write(NewModelEvent("update", s))
	return nil
}

func (sd *MockServiceDriver) Listen() chan *ModelEvent {
	return FromInterfaceChannel(sd.events.Listen())
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
		model,_ = NewArkenModel(sd, pd)

		Convey("When i create a service", func() {
			initialCreateCount := sd.calls["create"]
			service := &Service{}
			service.Init()

			service.Name = "testService"

			service, err := model.CreateService(service, false)

			Convey("Then the service should be available in all services", func() {
				So(err, ShouldBeNil)
				So(len(model.Services), ShouldEqual, 1)
				sc := model.Services["testService"]
				So(sc, ShouldNotBeNil)
			})

			Convey("Then its status should be stopped", func() {

				sc := model.Services["testService"]
				_, status := sc.Next()
				if st, ok := status.(StatusError); ok {
					So(st.ComputedStatus, ShouldEqual, STOPPED_STATUS)
				} else {
					So(ok, ShouldBeTrue)
				}
			})

			Convey("Then the service should be created in the backend", func() {

				So(sd.calls["create"], ShouldEqual, initialCreateCount+1)
				instance := model.Services["testService"].GetInstances()[0]
				So(instance.Config, ShouldNotBeNil)
				So(instance.Config.RancherInfo, ShouldNotBeNil)
				So(instance.Config.RancherInfo.EnvironmentId, ShouldEqual, "rancherId")
			})

			Convey("When I start the service", func() {
				initialStartCount := sd.calls["start"]
				model.StartService(service)

				Convey("Then the service should be started in the backend", func() {
					So(sd.calls["start"], ShouldEqual, initialStartCount+1)
				})

				Convey("Then its status should be starting", func() {
					So(getServiceStatus(model, "testService"), ShouldEqual, STARTING_STATUS)
				})

			})

			Convey("When I start the service and the service is started", func() {
				model.StartService(service)
				service.Status.Current = STARTED_STATUS
				service.Status.Alive = "1"

				Convey("Then its status should be started", func() {
					So(getServiceStatus(model, "testService"), ShouldEqual, STARTED_STATUS)

				})

			})

		})

	})
}

func getServiceStatus(model *Model, serviceName string) string {
	sc := model.Services[serviceName]
	s, status := sc.Next()
	if st, ok := status.(StatusError); ok {
		return st.ComputedStatus
	} else {
		return s.Status.Current
	}
}

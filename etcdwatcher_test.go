package goarken

import (
	"encoding/json"
	"errors"
	"github.com/coreos/go-etcd/etcd"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

func Test_EtcdWatcher(t *testing.T) {
	if os.Getenv("IT_Test") != "" {
		IT_EtcdWatcher(t)
	}
}

func IT_EtcdWatcher(t *testing.T) {

	client := etcd.NewClient([]string{})

	client.Delete("/domains", true)
	client.Delete("/services", true)

	var w *Watcher
	var updateChan chan interface{}

	Convey("Given a Watcher", t, func() {
		domains := make(map[string]*Domain)
		services := make(map[string]*ServiceCluster)

		w = &Watcher{
			Client:        client,
			DomainPrefix:  "/domains",
			ServicePrefix: "/services",
			Domains:       domains,
			Services:      services,
		}

		w.Init()
		updateChan = w.Listen()

		Convey("When it is started", func() {

			Convey("It doesn't contains any domain", func() {
				So(len(w.Domains), ShouldEqual, 0)
			})

			Convey("It doesn't contains any service", func() {
				So(len(w.Services), ShouldEqual, 0)
			})
		})

		Convey("When I add a domain", func() {

			_, err := client.Set("/domains/mydomain.com/type", "service", 0)
			if err != nil {
				panic(err)
			}

			So(len(w.Domains), ShouldEqual, 0)
			_, err = client.Set("/domains/mydomain.com/value", "my_service", 0)

			serviceOrDomain, err := wait(updateChan)
			So(serviceOrDomain, ShouldNotBeNil)

			if d, ok := serviceOrDomain.(*Domain); ok {
				So(d.Typ, ShouldEqual, "service")
				So(d.Value, ShouldEqual, "my_service")
			} else {
				So("tot", ShouldEqual, "error")
			}

			if err != nil {
				panic(err)
			}
			So(len(w.Domains), ShouldEqual, 1)
			domain := w.Domains["mydomain.com"]
			So(domain.Typ, ShouldEqual, "service")
			So(domain.Value, ShouldEqual, "my_service")

		})

		Convey("When I remove the domain in etcd", func() {
			_, err := client.Delete("/domains/mydomain.com", true)
			if err != nil {
				panic(err)
			}
			wait(updateChan)
			Convey("Then the domain is removed from the list of domains", func() {
				So(len(w.Domains), ShouldEqual, 0)

			})

		})

		Convey("When I add a service", func() {
			client.Set("/services/my_service/1/domain", "mydomain.com", 0)
			wait(updateChan)

			Convey("Then the status should be nil", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.(StatusError).Status, ShouldBeNil)

			})

		})

		// Creates a service that has not status, meaning n/a by default
		Convey("When I add a location to the service", func() {

			b, _ := json.Marshal(&Location{Host: "127.0.0.1", Port: 8080})
			client.Set("/services/my_service/1/location", string(b[:]), 0)
			wait(updateChan)

			Convey("Then it should be in n/a error", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, NA_STATUS)
			})
		})

		// When we create a service, it should be stopped
		Convey("When i add a stopped status", func() {
			client.Set("/services/my_service/1/status/expected", "stopped", 0)
			client.Set("/services/my_service/1/status/current", "stopped", 0)
			wait(updateChan)
			wait(updateChan)

			Convey("Then it should be stopped", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.(StatusError).ComputedStatus, ShouldEqual, STOPPED_STATUS)
			})

		})

		Convey("When I add an expected started status", func() {
			client.Set("/services/my_service/1/status/expected", "started", 0)
			wait(updateChan)
			Convey("Then the service should be in error (meaning unit has not set starting as current status)", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.(StatusError).ComputedStatus, ShouldEqual, ERROR_STATUS)
			})
		})

		Convey("When I add a current starting status", func() {
			client.Set("/services/my_service/1/status/current", "starting", 0)
			wait(updateChan)
			Convey("Then the service should be in starting", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.(StatusError).ComputedStatus, ShouldEqual, STARTING_STATUS)
			})
		})

		Convey("When I add a current started status", func() {
			client.Set("/services/my_service/1/status/current", "started", 0)
			wait(updateChan)
			Convey("Then the service should be in error (if not alive it should be in error)", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.(StatusError).ComputedStatus, ShouldEqual, ERROR_STATUS)
			})
		})

		Convey("When I add an alive key", func() {
			client.Set("/services/my_service/1/status/current", "started", 0)
			client.Set("/services/my_service/1/status/alive", "1", 0)
			wait(updateChan)
			Convey("Then the service should be starting", func() {
				service, err := w.Services["my_service"].Next()
				So(err, ShouldBeNil)
				So(service.Status.Compute(), ShouldEqual, STARTED_STATUS)
			})
		})

		Convey("When I passivate the service", func() {
			client.Set("/services/my_service/1/status/current", STOPPED_STATUS, 0)
			client.Set("/services/my_service/1/status/expected", PASSIVATED_STATUS, 0)
			wait(updateChan)
			Convey("Then the service should be starting", func() {
				_, err := w.Services["my_service"].Next()
				So(err, ShouldNotBeNil)
				So(err.(StatusError).ComputedStatus, ShouldEqual, PASSIVATED_STATUS)
			})
		})

		Convey("When I remove the service", func() {
			_, err := client.Delete("/services/my_service", true)
			if err != nil {
				panic(err)
			}
			waitFor(updateChan, time.Duration(3)*time.Second)
			Convey("Then the domain is removed from the list of domains", func() {
				So(len(w.Services), ShouldEqual, 0)

			})

		})

	})

}

func waitFor(iChannel chan interface{}, timeout time.Duration) (interface{}, error) {

	ticker := time.NewTicker(timeout)
	for {
		select {
		case <-ticker.C:
			return nil, errors.New("Timeout getting object")
		case i := <-iChannel:
			return i, nil
		}
	}
}

func wait(iChannel chan interface{}) (interface{}, error) {
	return waitFor(iChannel, time.Duration(3)*time.Second)
}

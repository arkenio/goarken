package storage

import (
	"fmt"
	"github.com/arkenio/goarken/model"
	. "github.com/arkenio/goarken/model"
	"github.com/coreos/go-etcd/etcd"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"sync"
)

func wait(listening chan *ModelEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	<- listening

}

func Test_EtcdWatcher(t *testing.T) {
	if os.Getenv("IT_Test") != "" {
		IT_EtcdWatcher(t)
	}
}

func IT_EtcdWatcher(t *testing.T) {
	testServiceName := "testService"

	client := etcd.NewClient([]string{})

	var w *Watcher
	var updateChan chan *model.ModelEvent

	notifCount := 0

	Convey("Given a Model with one service", t, func() {
		client.Delete("/domains", true)
		client.Delete("/services", true)

		w = NewWatcher(client, "/services", "/domains")
		updateChan = w.Listen()

		go func() {
			for {
				select {
				case <-updateChan:
					notifCount = notifCount + 1
				}
			}
		}()

		service := &Service{Name: testServiceName}
		service.Init()

		w.PersistService(service)

		Convey("When i get an etcd node from that service", func() {
			resp, _ := client.Get(fmt.Sprintf("/services/%s/1/status/expected", testServiceName), true, true)
			Convey("Then it can get the env key from it", func() {
				So(resp, ShouldNotBeNil)
				name, error := getEnvForNode(resp.Node)
				So(error, ShouldBeNil)
				So(name, ShouldEqual, testServiceName)
			})

		})

		Convey("When I create a Service", func() {

			Convey("Then the list of service contains 1 service", func() {
				services := w.LoadAllServices()
				So(len(services), ShouldEqual, 1)
			})

			Convey("Then it is able to load that service", func() {
				sc := w.LoadService(testServiceName)
				So(sc, ShouldNotBeNil)
			})

		})

		Convey("When i destroy a service", func() {
			nbServices := len(w.LoadAllServices())

			sc := w.LoadService(testServiceName)

			w.DestroyService(sc)

			Convey("Then the service should be destroyed", func() {
				sc := w.LoadService("serviceToBeDestroyed")
				So(sc, ShouldBeNil)
			})

			Convey("Then the number of services should have decreased", func() {
				So(len(w.LoadAllServices()), ShouldEqual, nbServices-1)
			})
		})

		Convey("When i modify a service", func() {

			initialNotifCount := notifCount
			sc := w.LoadService(testServiceName)

			service = sc.Instances[0]

			service.Status.Expected = STARTED_STATUS
			service.Config.RancherInfo = &RancherInfoType{EnvironmentId: "bla"}


			w.PersistService(service)

			Convey("Then the service should be modified", func() {
				sc := w.LoadService(testServiceName)
				service = sc.Instances[0]
				So(service.Status.Expected, ShouldEqual, STARTED_STATUS)
				So(service.Config.RancherInfo.EnvironmentId, ShouldEqual, "bla")
			})

			Convey("Then notification should have been sent", func() {
				So(notifCount, ShouldBeGreaterThan, initialNotifCount)
			})

		})
	})

	Convey("Given a Model with one domain", t, func() {
		client.Delete("/domains", true)
		client.Delete("/services", true)

		w = NewWatcher(client, "/services", "/domains")
		updateChan = w.Listen()

		go func() {
			for {
				select {
				case <-updateChan:
					notifCount = notifCount + 1
				}
			}
		}()

		domain := &Domain{}
		domain.Name = "test.domain.com"
		domain.Typ = "service"
		domain.Value = "testService"

		w.PersistDomain(domain)

		Convey("They key of that domain shoulde be well computed", func() {
			So(computeDomainNodeKey(domain.Name, "domains"), ShouldEqual, "/domains/test.domain.com")
		})

		Convey("When i get an etcd node from that domain", func() {
			resp, _ := client.Get(fmt.Sprintf("/domains/%s/type", domain.Name), true, true)
			So(resp, ShouldNotBeNil)
			Convey("Then it can get the env key from it", func() {
				name, _ := getDomainForNode(resp.Node)
				So(name, ShouldEqual, domain.Name)
			})

		})

		Convey("When it load all domains", func() {
			domains := w.LoadAllDomains()
			Convey("Then there should be one domain", func() {
				So(len(domains), ShouldEqual, 1)
			})

		})

		Convey("When it load one domain", func() {
			domain = w.LoadDomain(domain.Name)
			Convey("Then it should be equals to created domain", func() {
				So(domain, ShouldNotBeNil)
				So(domain.Name, ShouldEqual, "test.domain.com")
				So(domain.Value, ShouldEqual, "testService")
				So(domain.Typ, ShouldEqual, "service")
			})
		})

		Convey("When it destroy a domain", func() {
			w.DestroyDomain(domain)
			domains := w.LoadAllDomains()
			So(len(domains), ShouldEqual, 0)
		})

		Convey("When i modify a domain", func() {

			initialNotifCount := notifCount
			domain := w.LoadDomain("test.domain.com")

			domain.Value = "testService2"

			w.PersistDomain(domain)

			Convey("Then the domain should be modified", func() {
				domain := w.LoadDomain("test.domain.com")
				So(domain, ShouldNotBeNil)
				So(domain.Value, ShouldEqual, "testService2")
			})

			Convey("Then notifications	 should have been sent", func() {
				So(notifCount, ShouldBeGreaterThan, initialNotifCount)
			})

		})

		Convey("When two objet listens for event", func() {
			domain := w.LoadDomain("test.domain.com")
			domain.Value = "testService2"

			var wg sync.WaitGroup
			wg.Add(2)
			go wait(w.Listen(), &wg)
			go wait(w.Listen(), &wg)

			w.PersistDomain(domain)

			Convey("Then both object should have been notified", func() {
				wg.Wait()
				So(true, ShouldBeTrue)
			})

		})
	})

}

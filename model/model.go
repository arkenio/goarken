package model

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"sort"
	"time"
)

var log = logrus.New()

type Model struct {
	serviceDriver     ServiceDriver
	persistenceDriver PersistenceDriver

	Domains        map[string]*Domain
	Services       map[string]*ServiceCluster
	eventBroadcast *Broadcaster
	eventBuffer    *eventBuffer
}

func NewArkenModel(sDriver ServiceDriver, pDriver PersistenceDriver) (*Model, error) {
	if pDriver == nil {
		return nil, errors.New("Can't use a nil persistence Driver for Arken model")
	}

	model := &Model{
		Domains:           make(map[string]*Domain),
		Services:          make(map[string]*ServiceCluster),
		serviceDriver:     sDriver,
		persistenceDriver: pDriver,
		eventBroadcast:    NewBroadcaster(),
	}
	model.eventBuffer = newEventBuffer(model.eventBroadcast)
	err := model.Init()



	if err == nil {
		return model, nil
	} else {
		return nil, err
	}
}

func (m *Model) Listen() chan *ModelEvent {
	return FromInterfaceChannel(m.eventBroadcast.Listen())
}

func (m *Model) Init() error {

	//Load initial data
	domains, err := m.persistenceDriver.LoadAllDomains()
	if err != nil {
		return err
	}
	m.Domains = domains

	services, err := m.persistenceDriver.LoadAllServices()
	if err != nil {
		return err
	}
	m.Services = services

	go m.handlePersistenceModelEventOn(m.persistenceDriver.Listen())

	if m.serviceDriver != nil {
		go m.handlePersistenceModelEventOn(m.serviceDriver.Listen())
	}
	go m.eventBuffer.run()

	return nil

}

func (m *Model) CreateService(service *Service, startOnCreate bool) (*Service, error) {

	s, err := m.persistenceDriver.PersistService(service)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to persist service %s in etcd : %s", service.Name, err.Error()))
	}

	if m.serviceDriver != nil {
		info, err := m.serviceDriver.Create(s, startOnCreate)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Unable to create service in backend : %s", s.Name, err.Error()))
		}

		m.updateInfoFromDriver(s, info)
	}

	s, err = m.saveService(s)
	if err != nil {
		return nil, err
	}

	if s.Domain != "" {

		if domain, ok := m.Domains[s.Domain]; ok {
			domain.Typ = "service"
			domain.Value = s.Name
			_, err = m.UpdateDomain(domain)
			if err != nil {
				log.Errorf("Unable to update domain %s for service %s : %v", s.Domain, s.Name, err)
			}
		} else {
			_, err := m.CreateDomain(&Domain{Name: s.Domain, Typ: "service", Value: s.Name})
			if err != nil {
				log.Errorf("Unable to create domain %s for service %s : %v", s.Domain, s.Name, err)
			}
		}
	}

	m.eventBuffer.events <- NewModelEvent("create", s)

	return s, nil

}

func (m *Model) CreateDomain(domain *Domain) (*Domain, error) {
	domain, err := m.persistenceDriver.PersistDomain(domain)
	if err != nil {
		return nil, err
	} else {
		m.eventBuffer.events <- NewModelEvent("create", domain)
		return domain, nil
	}
}

func (m *Model) DestroyDomain(domain *Domain) error {

	err := m.persistenceDriver.DestroyDomain(domain)
	if err != nil {
		return err
	} else {
		m.eventBuffer.events <- NewModelEvent("delete", domain)
		return nil
	}

}

func (m *Model) UpdateDomain(domain *Domain) (*Domain, error) {
	domain, err := m.persistenceDriver.PersistDomain(domain)
	if err != nil {
		return nil, err
	} else {
		m.eventBuffer.events <- NewModelEvent("update", domain)
		return domain, nil
	}
}

func (m *Model) StartService(service *Service) (*Service, error) {

	if m.serviceDriver != nil {
		info, err := m.serviceDriver.Start(service)
		if err != nil {
			return nil, err
		}
		m.updateInfoFromDriver(service, info)
	}

	service.Status.Expected = STARTED_STATUS
	service.Status.Current = STARTING_STATUS
	service, err := m.saveService(service)

	if err != nil {
		return nil, err
	} else {
		m.eventBuffer.events <- NewModelEvent("update", service)
		return service, nil
	}

}

func (m *Model) StopService(service *Service) (*Service, error) {
	service.Status.Expected = STOPPED_STATUS
	if m.serviceDriver != nil {
		info, err := m.serviceDriver.Stop(service)
		if err != nil {
			return nil, err
		}

		m.updateInfoFromDriver(service, info)
	}

	service, err := m.saveService(service)

	if err != nil {
		return nil, err
	} else {
		m.eventBuffer.events <- NewModelEvent("update", service)
		return service, nil
	}
}

func (m *Model) PassivateService(service *Service) (*Service, error) {
	service.Status.Expected = PASSIVATED_STATUS

	info, err := m.serviceDriver.Stop(service)
	if err != nil {
		return nil, err
	}

	m.updateInfoFromDriver(service, info)

	service, err = m.saveService(service)

	if err != nil {
		return nil, err
	} else {
		m.eventBuffer.events <- NewModelEvent("update", service)
		return service, nil
	}
}

func (m *Model) DestroyService(service *Service) error {
	if m.serviceDriver != nil {
		err := m.serviceDriver.Destroy(service)
		if err != nil {
			return err
		}
	}

	error := m.persistenceDriver.DestroyService(m.Services[service.Name])
	if error != nil {
		return error
	} else {
		m.eventBuffer.events <- NewModelEvent("delete", service)
		return nil
	}
}

func (m *Model) DestroyServiceCluster(sc *ServiceCluster) error {
	for _, service := range sc.Instances {
		if m.serviceDriver != nil {
			err := m.serviceDriver.Destroy(service)
			if err != nil {
				return err
			} else {
				m.eventBuffer.events <- NewModelEvent("delete", service)
			}
		}
	}

	return m.persistenceDriver.DestroyService(sc)
}

func (m *Model) saveService(service *Service) (*Service, error) {

	return m.persistenceDriver.PersistService(service)
}

func (m *Model) updateInfoFromDriver(service *Service, info interface{}) {
	if rancherInfo, ok := info.(*RancherInfoType); ok {
		service.Config.RancherInfo = rancherInfo
	}

	if fleetInfo, ok := info.(*FleetInfoType); ok {
		service.Config.FleetInfo = fleetInfo
	}
	m.eventBuffer.events <- NewModelEvent("update", service)

}

type eventBuffer struct {
	eventsMap      map[string]*ModelEvent
	events         chan *ModelEvent
	eventBroadcast *Broadcaster
}

func newEventBuffer(b *Broadcaster) *eventBuffer {
	return &eventBuffer{
		eventsMap:      make(map[string]*ModelEvent),
		events:         make(chan *ModelEvent),
		eventBroadcast: b,
	}
}

func (eb *eventBuffer) run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			events := make([]*ModelEvent,0, len(eb.eventsMap))
			for _,v := range eb.eventsMap {
				events = append(events, v)
			}
			sort.Sort(ModelByTime(events))

			for _,event := range events {
				eb.eventBroadcast.Write(event)
				delete(eb.eventsMap, eb.keyFromModelEvent(event))
			}
			break
		case event := <-eb.events:
			eb.eventsMap[eb.keyFromModelEvent(event)] = event
			break
		}
	}
}

func (eb *eventBuffer) keyFromModelEvent(event *ModelEvent) string {
	if sc, ok := event.Model.(*ServiceCluster); ok {
		return fmt.Sprintf("SC_%s_%s", event.EventType, sc.Name)
	} else if domain, ok := event.Model.(*Domain); ok {
		return fmt.Sprintf("D_%s_%s", event.EventType, domain.Name)
	} else if service, ok := event.Model.(*Service); ok {
		return fmt.Sprintf("S_%s_%s", event.EventType, service.Name)
	}
	return "unknown"
}

func (m *Model) handlePersistenceModelEventOn(eventStream chan *ModelEvent) {

	for {
		event := <-eventStream

		switch event.EventType {
		case "create":
		case "update":
			if sc, ok := event.Model.(*ServiceCluster); ok {
				m.Services[sc.Name] = sc
				m.eventBuffer.events <- event
			} else if domain, ok := event.Model.(*Domain); ok {
				m.Domains[domain.Name] = domain
				m.eventBuffer.events <- event
			} else if info, ok := event.Model.(*RancherInfoType); ok {
				m.onRancherInfo(info)
			}

		case "delete":
			if sc, ok := event.Model.(*ServiceCluster); ok {
				delete(m.Services, sc.Name)
				m.eventBuffer.events <- event
			} else if domain, ok := event.Model.(*Domain); ok {
				delete(m.Domains, domain.Name)
			}
			m.eventBuffer.events <- event
		}



	}

}

func (m *Model) onRancherInfo(info *RancherInfoType) {
	sc := m.Services[info.EnvironmentName]
	if sc != nil {
		for _, service := range sc.GetInstances() {
			service.Config.RancherInfo = info

			if !service.Location.Equals(info.Location) {
				log.Infof("Service %s changed location from %s to %s", service.Name, service.Location, info.Location)
				service.Location = info.Location

			}

			// Save last status
			computedSatus := service.Status.Compute()

			service.Status.Current = info.CurrentStatus
			//If service is stopped it may be passivated
			if info.CurrentStatus == STOPPED_STATUS && service.Status.Expected == PASSIVATED_STATUS {
				service.Status.Current = PASSIVATED_STATUS
			}

			if service.Status.Current == STARTED_STATUS {
				service.Status.Alive = "1"
			} else {
				service.Status.Alive = ""
			}

			// Compare to initial status
			if computedSatus != service.Status.Compute() {
				log.Infof("Service %s changed its status to : %s", service.Name, service.Status.Compute())
			}

			s, err := m.persistenceDriver.PersistService(service)



			if err != nil {
				log.Errorf("Error when persisting rancher update : %s", err.Error())
				log.Errorf("Rancher update was : %s", info)
			} else {
				m.eventBuffer.events <- NewModelEvent("update", s)
			}

		}
	}
}

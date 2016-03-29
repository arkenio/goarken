package model

import (
	"fmt"
)

type Model struct {
	serviceDriver     ServiceDriver
	persistenceDriver PersistenceDriver

	Domains  map[string]*Domain
	Services map[string]*ServiceCluster
}

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

func FromInterfaceChannel(fromChannel chan interface{}, toChannel chan *ModelEvent) {

	go func() {
		for {
			event := <-fromChannel
			if evt, ok := event.(*ModelEvent); ok {
				toChannel <- evt
			} else {
				panic(event)
			}

		}
	}()

}

func NewArkenModel(sDriver ServiceDriver, pDriver PersistenceDriver) *Model {
	if pDriver == nil {
		panic("Can't use a nil persistence Driver")
	}

	model := &Model{
		Domains:           make(map[string]*Domain),
		Services:          make(map[string]*ServiceCluster),
		serviceDriver:     sDriver,
		persistenceDriver: pDriver,
	}

	model.Init()

	return model
}

func (m *Model) Init() {

	//Load initial data
	m.Domains = m.persistenceDriver.LoadAllDomains()
	m.Services = m.persistenceDriver.LoadAllServices()

	go func() {
		m.handlePersitenceModelEventOn(m.persistenceDriver.Listen())
	}()

	//	if m.serviceDriver != nil {
	//		go m.handleModelEventOn(m.serviceDriver.Listen())
	//	}

}

func (m *Model) Create(s *Service, startOnCreate bool) (*Service, error) {

	m.persistenceDriver.PersistService(s)

	res, err := m.serviceDriver.Create(s, startOnCreate)
	if err != nil {
		return nil, err
	}

	if info, ok := res.(RancherInfoType); ok {
		s.Config.RancherInfo = info
	}

	if info, ok := res.(FleetInfoType); ok {
		s.Config.FleetInfo = info
	}

	m.persistenceDriver.PersistService(s)

	return s, nil
}

func (m *Model) handlePersitenceModelEventOn(eventStream chan *ModelEvent) {

	for {
		event := <-eventStream

		switch event.eventType {
		case "create":
		case "update":
			if sc, ok := event.model.(*ServiceCluster); ok {
				m.Services[sc.Name] = sc
			} else if domain, ok := event.model.(Domain); ok {
				m.Domains[domain.Name] = &domain
			}
		case "delete":
			if sc, ok := event.model.(ServiceCluster); ok {
				delete(m.Services, sc.Name)

			} else if domain, ok := event.model.(Domain); ok {
				delete(m.Domains, domain.Name)
			}
		}

	}

}

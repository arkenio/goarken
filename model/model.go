package model

import "time"

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

func NewArkenModel(domainPrefix string, servicePrefix string, sDriver ServiceDriver, pDriver PersistenceDriver) *Model {
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

	go m.handleModelEventOn(m.persistenceDriver.Listen())

	if m.serviceDriver != nil {
		go m.handleModelEventOn(m.serviceDriver.Listen())
	}


}

func (m *Model) handleModelEventOn(eventStream chan ModelEvent) {
	for {
		select {
		case event := <-eventStream:
			switch event.eventType {
			case "create":
			case "update":
				if sc, ok := event.model.(ServiceCluster); ok {
					m.Services[sc.Name] = sc
				} else if domain, ok := event.model.(Domain); ok {
					m.Domains[domain.Name] = domain
				}
			case "delete":
				if sc, ok := event.model.(ServiceCluster); ok {
					delete(m.Services, sc.Name)

				} else if domain, ok := event.model.(Domain); ok {
					delete(m.Domains, domain.Name)
				}
			}
		default:
			time.Sleep(time.Second)
		}
	}

}

package model

type Model struct {
	serviceDriver     ServiceDriver
	persistenceDriver PersistenceDriver

	Domains  map[string]*Domain
	Services map[string]*ServiceCluster
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
		m.handlePersistenceModelEventOn(m.persistenceDriver.Listen())
	}()

}

func (m *Model) CreateService(s *Service, startOnCreate bool) (*Service, error) {

	s, err := m.persistenceDriver.PersistService(s)
	if err != nil {
		return nil,err
	}

	if m.serviceDriver != nil {
		info, err := m.serviceDriver.Create(s, startOnCreate)
		if err != nil {
			return nil, err
		}

		m.updateInfoFromDriver(s, info)
	}

	return m.saveService(s)

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
	return m.saveService(service)

}

func (m *Model) StopService(service *Service) (*Service, error) {
	if m.serviceDriver != nil {
		info, err := m.serviceDriver.Stop(service)
		if err != nil {
			return nil, err
		}

		m.updateInfoFromDriver(service, info)
	}

	return m.saveService(service)
}

func (m *Model) DestroyService(service *Service) error {
	if m.serviceDriver != nil {
		err := m.serviceDriver.Destroy(service)
		if err != nil {
			return err
		}
	}

	return m.persistenceDriver.DestroyService(m.Services[service.Name])
}

func (m *Model) DestroyServiceCluster(sc *ServiceCluster) error {
	for _, service := range sc.Instances {
		if m.serviceDriver != nil {
			err := m.serviceDriver.Destroy(service)
			if err != nil {
				return err
			}
		}
	}


	return m.persistenceDriver.DestroyService(sc)
}



func (m *Model) saveService(service *Service) (*Service, error) {
	service, err := m.persistenceDriver.PersistService(service)
	if err == nil {
		m.Services[service.Name].Add(service)
		return service, nil
	} else {
		return nil, err
	}
}

func (m *Model) updateInfoFromDriver(service *Service, info interface{}) {
	if rancherInfo, ok := info.(*RancherInfoType); ok {
		service.Config.RancherInfo = rancherInfo
	}

	if fleetInfo, ok := info.(*FleetInfoType); ok {
		service.Config.FleetInfo = fleetInfo
	}

}

func (m *Model) handlePersistenceModelEventOn(eventStream chan *ModelEvent) {

	for {
		event := <-eventStream

		switch event.eventType {
		case "create":
		case "update":
			if sc, ok := event.model.(*ServiceCluster); ok {
				m.Services[sc.Name] = sc
			} else if domain, ok := event.model.(*Domain); ok {
				m.Domains[domain.Name] = domain
			}
		case "delete":
			if sc, ok := event.model.(*ServiceCluster); ok {
				delete(m.Services, sc.Name)

			} else if domain, ok := event.model.(*Domain); ok {
				delete(m.Domains, domain.Name)
			}
		}

	}

}

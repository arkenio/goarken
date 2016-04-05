package model
import (
	"github.com/Sirupsen/logrus"
	"errors"
	"fmt"
)


var log = logrus.New()

type Model struct {
	serviceDriver     ServiceDriver
	persistenceDriver PersistenceDriver

	Domains  map[string]*Domain
	Services map[string]*ServiceCluster
}

func NewArkenModel(sDriver ServiceDriver, pDriver PersistenceDriver) (*Model,error) {
	if pDriver == nil {
		return nil, errors.New("Can't use a nil persistence Driver for Arken model")
	}

	model := &Model{
		Domains:           make(map[string]*Domain),
		Services:          make(map[string]*ServiceCluster),
		serviceDriver:     sDriver,
		persistenceDriver: pDriver,
	}

	err := model.Init()

	if err == nil {
		return model,nil
	} else {
		return nil, err
	}
}

func (m *Model) Init() error{

	//Load initial data
	domains , err := m.persistenceDriver.LoadAllDomains()
	if err != nil {
		return err
	}
	m.Domains = domains


	services, err := m.persistenceDriver.LoadAllServices()
	if err != nil {
		return err
	}
	m.Services = services

	go func() {
		m.handlePersistenceModelEventOn(m.persistenceDriver.Listen())
	}()

	go func() {
		m.handlePersistenceModelEventOn(m.serviceDriver.Listen())
	}()

	return nil

}

func (m *Model) CreateService(s *Service, startOnCreate bool) (*Service, error) {

	s, err := m.persistenceDriver.PersistService(s)
	if err != nil {
		return nil,errors.New(fmt.Sprintf("Unable to persist service %s in etcd : %s",s.Name, err.Error()))
	}

	if m.serviceDriver != nil {
		info, err := m.serviceDriver.Create(s, startOnCreate)
		if err != nil {
			return nil,errors.New(fmt.Sprintf("Unable to create service in backend : %s",s.Name, err.Error()))
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
			} else if info, ok := event.model.(*RancherInfoType); ok {
				m.onRancherInfo(info)
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

func (m *Model) onRancherInfo(info *RancherInfoType) {
	sc := m.Services[info.EnvironmentName]
	if sc != nil {
		for _,service := range sc.GetInstances() {
			service.Config.RancherInfo = info

			if(!service.Location.Equals(info.Location)) {
				log.Infof("Service %s changed location from %s to %s", service.Name, service.Location, info.Location)
				service.Location = info.Location

			}

			computedSatus := service.Status.Compute()
			service.Status.Current = info.CurrentStatus
			if(service.Status.Current == STARTED_STATUS) {
				service.Status.Alive = "1"
			} else {
				service.Status.Alive = ""
			}
			if(computedSatus != service.Status.Compute()) {
				log.Infof("Service %s changed its status to : %s", service.Name, service.Status.Compute())
			}

			_, err := m.persistenceDriver.PersistService(service)
			if(err!= nil) {
				log.Errorf("Error when persisting rancher update : %s", err.Error())
				log.Errorf("Rancher update was : %s" , info)
			}
		}
	}
}

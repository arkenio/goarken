package model

type ServiceDriver interface {
	Create(s *Service, startOnCreate bool) (interface{}, error)
	Start(s *Service) (interface{}, error)
	Stop(s *Service) (interface{}, error)
	Destroy(s *Service) error

	Listen() chan *ModelEvent
}

type PersistenceDriver interface {
	LoadAllServices() (map[string]*ServiceCluster, error)
	LoadService(serviceName string) (*ServiceCluster, error)
	PersistService(*Service) (*Service, error)
	DestroyService(*ServiceCluster) error

	LoadAllDomains() (map[string]*Domain, error)
	LoadDomain(serviceName string) (*Domain, error)
	PersistDomain(*Domain) (*Domain, error)
	DestroyDomain(*Domain) error

	Listen() chan *ModelEvent
}

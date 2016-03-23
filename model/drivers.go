package model




type ServiceDriver interface {
	Create(s *Service, startOnCreate bool) (interface{},error)
	Start(s *Service) (interface{},error)
	Stop(s *Service) (interface{},error)
	Destroy(s *Service) error

	Listen() chan *ModelEvent
}


type PersistenceDriver interface  {
	LoadAllServices() map[string]*ServiceCluster
	LoadService(serviceName string) *ServiceCluster
	PersistService(*Service) (*Service, error)
	DestroyService(*ServiceCluster) error

	LoadAllDomains() map[string]*Domain
	LoadDomain(serviceName string) *Domain
	PersistDomain(*Domain) (*Domain, error)
	DestroyDomain(*Domain) error

	Listen() chan *ModelEvent

}


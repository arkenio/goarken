package model




type ServiceDriver interface {
	Create(s *Service, startOnCreate bool) (*Service,error)
	Start(s *Service) (*Service,error)
	Stop(s *Service) (*Service,error)
	Destroy(s *Service) error

	Listen() chan ModelEvent
}


type PersistenceDriver interface  {
	LoadAllServices() map[string]*ServiceCluster
	LoadService(serviceName string) *Service
	PersistService(*Service) (*Service, error)
	DestroyService(*Service) error

	LoadAllDomains() map[string]*Domain
	LoadDomain(serviceName string) *Domain
	PersistDomain(*Domain) (*Service, error)
	DestroyDomain(*Domain) error

	Listen() chan ModelEvent

}


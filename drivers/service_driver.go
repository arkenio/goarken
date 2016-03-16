package drivers
import . "github.com/arkenio/goarken"




type ServiceDriver interface {
	Create(s *Service) (*Service,error)
	Start(s *Service) (*Service,error)
	Stop(s *Service) (*Service,error)
	Passivate(s *Service) (*Service,error)
	Destroy(s *Service) error
}


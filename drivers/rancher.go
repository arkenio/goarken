package drivers

import (
	"errors"
	. "github.com/arkenio/goarken"
	"github.com/coreos/go-etcd/etcd"
)

type RancherServiceDriver struct {
	client           *etcd.Client
	rancherHost      string
	rancherAccessKey string
	rancherSecretKey string
}

func NewRancherServiceDriver(client *etcd.Client,rancherHost string,rancherAccessKey string,rancherSecretKey string) *RancherServiceDriver {
	return &RancherServiceDriver{
		client,
		rancherHost,
		rancherAccessKey,
		rancherSecretKey,
	}
}

func (r *RancherServiceDriver) Create(s *Service) (*Service, error) {
	return s, errors.New("Not implemented")
}

func (r *RancherServiceDriver) Start(s *Service) (*Service, error) {
	return s, errors.New("Not implemented")
}

func (r *RancherServiceDriver) Stop(s *Service) (*Service, error) {
	return s, errors.New("Not implemented")
}

func (r *RancherServiceDriver) Passivate(s *Service) (*Service, error) {
	return s, errors.New("Not implemented")
}

func (r *RancherServiceDriver) Destroy(s *Service) error {
	return errors.New("Not implemented")
}

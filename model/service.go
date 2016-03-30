package model

import (
	"github.com/Sirupsen/logrus"
	"time"
)

type Location struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s *Location) Equals(other *Location) bool {
	if s == nil && other == nil {
		return true
	}

	return s != nil && other != nil &&
		s.Host == other.Host &&
		s.Port == other.Port
}

func (s *Location) IsFullyDefined() bool {
	if s == nil {
		return false
	}
	return s.Host != "" && s.Port != 0

}

type ServiceConfig struct {
	Robots      string `json:"robots"`
	Environment map[string]interface{}
	RancherInfo *RancherInfoType `json:"rancherInfo,omitempty"`
	FleetInfo   *FleetInfoType   `json:"fleetInfo,omitempty"`
}

type RancherInfoType struct {
	ServiceId string
}

type FleetInfoType struct {
	UnitName string
}

func (config *ServiceConfig) Equals(other *ServiceConfig) bool {
	if config == nil && other == nil {
		return true
	}

	return config != nil && other != nil &&
		config.Robots == other.Robots
}

type Service struct {
	Index      string         `json:"index"`
	NodeKey    string         `json:"nodeKey"`
	Location   *Location      `json:"location"`
	Domain     string         `json:"domain"`
	Name       string         `json:"name"`
	Status     *Status        `json:"status"`
	LastAccess *time.Time     `json:"lastAccess"`
	Config     *ServiceConfig `json:"config"`
	log        *logrus.Logger
}

func (s *Service) Init() *Service {

	s.Index = "1"

	status := NewInitialStatus(STOPPED_STATUS, s)
	s.Status = status
	s.Config = &ServiceConfig{}

	return s

}

func (service *Service) Equals(other *Service) bool {
	if service == nil && other == nil {
		return true
	}

	return service != nil && other != nil &&
		service.Location.Equals(other.Location) &&
		service.Status.Equals(other.Status) &&
		service.Config.Equals(other.Config)
}

func (s *Service) StartedSince() *time.Time {
	if s == nil {
		return nil
	}

	if s.Status != nil &&
		s.Status.Current == STARTED_STATUS {
		return s.LastAccess
	} else {
		return nil
	}
}

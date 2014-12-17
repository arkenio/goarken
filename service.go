package goarken

import (
	"strings"
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
	return s.Host != "" && s.Port != 0
}

type Service struct {
	Index      string
	NodeKey    string
	Location   *Location
	Domain     string
	Name       string
	Status     *Status
	LastAccess *time.Time
}

func (s *Service) UnitName() string {
	return "nxio@" + strings.Split(s.Name, "_")[1] + ".service"
}

func (service *Service) Equals(other *Service) bool {
	if service == nil && other == nil {
		return true
	}

	return service != nil && other != nil &&
		service.Location.Equals(other.Location) &&
		service.Status.Equals(other.Status)
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

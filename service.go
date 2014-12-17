package goarken

import "time"

type location struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s *location) equals(other *location) bool {
	if s == nil && other == nil {
		return true
	}

	return s != nil && other != nil &&
		s.Host == other.Host &&
		s.Port == other.Port
}

func (s *location) isFullyDefined() bool {
	return s.Host != "" && s.Port != 0
}

type Service struct {
	index      string
	nodeKey    string
	location   *location
	domain     string
	name       string
	status     *Status
	lastAccess *time.Time
}

func (service *Service) equals(other *Service) bool {
	if service == nil && other == nil {
		return true
	}

	return service != nil && other != nil &&
		service.location.equals(other.location) &&
		service.status.equals(other.status)
}

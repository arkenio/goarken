package goarken

import (
	"encoding/json"
	"errors"
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
	"regexp"
	"strconv"
	"strings"
	"time"
	"github.com/Sirupsen/logrus"
)

var (
	serviceRegexp = regexp.MustCompile("/services/(.*)(/.*)*")
)

type Location struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func SetServicePrefix(servicePrefix string) {
	serviceRegexp = regexp.MustCompile(servicePrefix + "/(.*)(/.*)*")
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

func getEnvIndexForNode(node *etcd.Node) string {
	return strings.Split(serviceRegexp.FindStringSubmatch(node.Key)[1], "/")[1]
}

func getEnvForNode(node *etcd.Node) string {
	return strings.Split(serviceRegexp.FindStringSubmatch(node.Key)[1], "/")[0]
}

type ServiceConfig struct {
	Robots string `json:"robots"`
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

func NewService(serviceNode *etcd.Node) (*Service, error) {

	serviceIndex := getEnvIndexForNode(serviceNode)

	if _, err := strconv.Atoi(serviceIndex); err != nil {
		// Don't handle node that are not integer (ie config node)
		return nil, errors.New("Not a service index node")
	}

	service := &Service{}
	service.log = logrus.New()
	service.Location = &Location{}
	service.Config = &ServiceConfig{Robots: ""}
	service.Index = getEnvIndexForNode(serviceNode)
	service.Name = getEnvForNode(serviceNode)
	service.NodeKey = serviceNode.Key

	for _, node := range serviceNode.Nodes {
		switch node.Key {
		case service.NodeKey + "/location":
			location := &Location{}
			err := json.Unmarshal([]byte(node.Value), location)
			if err == nil {
				service.Location.Host = location.Host
				service.Location.Port = location.Port
			}

		case service.NodeKey + "/config":
			for _, subNode := range node.Nodes {
				switch subNode.Key {
				case service.NodeKey + "/config/gogeta":
					serviceConfig := &ServiceConfig{}
					err := json.Unmarshal([]byte(subNode.Value), serviceConfig)
					if err == nil {
						service.Config = serviceConfig
					}
				}
			}

		case service.NodeKey + "/domain":
			service.Domain = node.Value
		case service.NodeKey + "/lastAccess":
			lastAccess := node.Value
			lastAccessTime, err := time.Parse(TIME_FORMAT, lastAccess)
			if err != nil {
				glog.Errorf("Error parsing last access date with service %s: %s", service.Name, err)
				break
			}
			service.LastAccess = &lastAccessTime

		case service.NodeKey + "/status":
			service.Status = NewStatus(service, node)
		}
	}
	return service, nil
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


package goarken

import (
	"encoding/json"
	"errors"
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	serviceRegexp = regexp.MustCompile("/services/(.*)(/.*)*")
)

type Location struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func setServicePrefix(servicePrefix string) {
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

type Service struct {
	Index      string
	NodeKey    string
	Location   *Location
	Domain     string
	Name       string
	Status     *Status
	LastAccess *time.Time
}

func NewService(serviceNode *etcd.Node) (*Service, error) {

	serviceIndex := getEnvIndexForNode(serviceNode)

	if _, err := strconv.Atoi(serviceIndex); err != nil {
		// Don't handle node that are not integer (ie config node)
		return nil, errors.New("Not a service index node")
	}

	service := &Service{}
	service.Location = &Location{}
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

func (s *Service) Start(client *etcd.Client) error {
	return s.fleetcmd("start", client)
}

func (s *Service) Stop(client *etcd.Client) error {
	return s.fleetcmd("destroy", client)
}

func (s *Service) Passivate(client *etcd.Client) error {
	err := s.fleetcmd("destroy", client)
	if err != nil {
		return err
	}

	statusKey := s.NodeKey + "/status"

	responseCurrent, error := client.Set(statusKey+"/current", PASSIVATED_STATUS, 0)
	if error != nil && responseCurrent == nil {
		glog.Errorf("Setting status current to 'passivated' has failed for Service "+s.Name+": %s", err)
	}

	response, error := client.Set(statusKey+"/expected", PASSIVATED_STATUS, 0)
	if error != nil && response == nil {
		glog.Errorf("Setting status expected to 'passivated' has failed for Service "+s.Name+": %s", err)
	}
	return nil
}

func (s *Service) fleetcmd(command string, client *etcd.Client) error {
	//TODO Use fleet's REST API
	etcdAddress := client.GetCluster()[0]
	cmd := exec.Command("/usr/bin/fleetctl", "--endpoint="+etcdAddress, command, s.UnitName())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

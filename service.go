package goarken

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
	"os"
	"os/exec"
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

func NewService(client *etcd.Client) *Service {
	return &Service{client: client}
}

type Service struct {
	client     *etcd.Client
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

func (s *Service) Start() error {
	return s.fleetcmd("start")
}

func (s *Service) Stop() error {
	return s.fleetcmd("destroy")
}

func (s *Service) Passivate() error {
	err := s.fleetcmd("destroy")
	if err != nil {
		return err
	}

	statusKey := s.NodeKey + "/status"

	// TODO could expose an API in goarken package
	responseCurrent, error := s.client.Set(statusKey+"/current", PASSIVATED_STATUS, 0)
	if error != nil && responseCurrent == nil {
		glog.Errorf("Setting status current to 'passivated' has failed for Service "+s.Name+": %s", err)
	}

	response, error := s.client.Set(statusKey+"/expected", PASSIVATED_STATUS, 0)
	if error != nil && response == nil {
		glog.Errorf("Setting status expected to 'passivated' has failed for Service "+s.Name+": %s", err)
	}
	return nil
}

func (s *Service) fleetcmd(command string) error {
	//TODO Use fleet's REST API
	etcdAddress := s.client.GetCluster()[0]
	cmd := exec.Command("/usr/bin/fleetctl", "--endpoint="+etcdAddress, command, s.UnitName())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

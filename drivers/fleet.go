package drivers



import (
	"github.com/coreos/go-etcd/etcd"
	. "github.com/arkenio/goarken/model"
	"os/exec"
	"os"
	"strings"
	"errors"
	"fmt"
	"github.com/golang/glog"
)




type FleetServiceDriver struct {
	client *etcd.Client
}


func NewFleetServiceDriver(client *etcd.Client) *FleetServiceDriver {
	return &FleetServiceDriver{client}
}

func (f *FleetServiceDriver) Create(s *Service, startOnCreate bool) (interface{},error) {
	return nil, errors.New("Not implemented")
}


func (f *FleetServiceDriver) Start(s *Service) (interface{},error) {
	err := f.fleetcmd(s, "start", f.client)
	return s,err
}

func (f *FleetServiceDriver) Stop(s *Service) (interface{},error) {
	err := f.fleetcmd(s, "stop", f.client)
	return s,err
}


func (f *FleetServiceDriver) Passivate(s *Service) (interface{},error) {
	glog.Info(fmt.Sprintf("Passivating service %s",s.Name))
	err := f.fleetcmd(s, "destroy", f.client)
	if err != nil {
		return s,err
	}

	statusKey := s.NodeKey + "/status"

	responseCurrent, error := f.client.Set(statusKey+"/current", PASSIVATED_STATUS, 0)
	if error != nil && responseCurrent == nil {
		glog.Errorf("Setting status current to 'passivated' has failed for Service "+s.Name+": %s", error)
	}

	response, error := f.client.Set(statusKey+"/expected", PASSIVATED_STATUS, 0)
	if error != nil && response == nil {
		glog.Errorf("Setting status expected to 'passivated' has failed for Service "+s.Name+": %s", error)
	}
	return s, nil
}

func (f *FleetServiceDriver) Destroy(s *Service)  error {
	err := f.fleetcmd(s, "destroy", f.client)
	return err
}

func  unitNameFromService(s *Service) string {
	return "nxio@" + strings.Split(s.Name, "_")[1] + ".service"
}


func (f *FleetServiceDriver) fleetcmd(s *Service, command string, client *etcd.Client) error {
	//TODO Use fleet's REST API
	etcdAddress := client.GetCluster()[0]

	cmd := exec.Command("/usr/bin/fleetctl", "--endpoint="+etcdAddress, command, unitNameFromService(s))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}


func UnitName(s *Service) string {
	return "nxio@" + strings.Split(s.Name, "_")[1] + ".service"
}


func (f *FleetServiceDriver) Listen() chan ModelEvent {
	return nil
}
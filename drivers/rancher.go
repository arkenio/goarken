package drivers

import (
	. "github.com/arkenio/goarken/model"
	"github.com/coreos/go-etcd/etcd"
	"github.com/rancher/go-rancher/client"
	"io/ioutil"
	//"encoding/json"
	"github.com/Sirupsen/logrus"
)

type RancherServiceDriver struct {
	etcdClient    *etcd.Client
	rancherClient *client.RancherClient
}


func NewRancherServiceDriver(etcdClient *etcd.Client, rancherHost string, rancherAccessKey string, rancherSecretKey string) (*RancherServiceDriver, error) {

	rancherClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       rancherHost,
		AccessKey: rancherAccessKey,
		SecretKey: rancherSecretKey,
	})

	if err != nil {
		return nil, err
	}

	return &RancherServiceDriver{
		etcdClient,
		rancherClient,
	},nil
}


func (r *RancherServiceDriver) Create(s *Service, startOnCreate bool) (interface{}, error) {

	//Start rancher environment
	env := &client.Environment{}
	env.StartOnCreate = startOnCreate
	env.Name = s.Name
	env.Environment = s.Config.Environment
	fillCompose(env)

	env, err := r.rancherClient.Environment.Create(env)

	check(err)

	return &RancherInfoType{ServiceId:env.Id}, nil

}


func fillCompose(env *client.Environment) {
	dat, err := ioutil.ReadFile("/Users/dmetzler/src/github.com/dmetzler/community-catalog/templates/nuxeo/0/docker-compose.yml")
	check(err)
	env.DockerCompose = string(dat)

	dat, err = ioutil.ReadFile("/Users/dmetzler/src/github.com/dmetzler/community-catalog/templates/nuxeo/0/rancher-compose.yml")
	check(err)
	env.RancherCompose = string(dat)

}

func (r *RancherServiceDriver) Start(s *Service) (interface{}, error) {
	rancherId := s.Config.RancherInfo.ServiceId
	env, err := r.rancherClient.Environment.ById(rancherId)
	check(err)
	env, err = r.rancherClient.Environment.ActionActivateservices(env)
	return s, err
}

func (r *RancherServiceDriver) Stop(s *Service) (interface{}, error) {
	rancherId := s.Config.RancherInfo.ServiceId
	env, err := r.rancherClient.Environment.ById(rancherId)
	check(err)
	env, err = r.rancherClient.Environment.ActionDeactivateservices(env)
	return s, err
}


func (r *RancherServiceDriver) Destroy(s *Service) error {
	rancherId := s.Config.RancherInfo.ServiceId
	env, err := r.rancherClient.Environment.ById(rancherId)
	check(err)
	env, err = r.rancherClient.Environment.ActionDeactivateservices(env)
	return err
}


func check(e error) {
	if e != nil {
		log := logrus.New()
		log.Info("error: ", e.Error())
		panic(e)
	}
}


func (r *RancherServiceDriver) Listen() chan ModelEvent {
	return nil
}
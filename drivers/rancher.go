package drivers

import (
	"encoding/base64"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	. "github.com/arkenio/goarken/model"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/go-rancher/client"
	"io/ioutil"
	"net/http"
	"net/url"
	"fmt"
)

var log = logrus.New()

type RancherServiceDriver struct {
	rancherClient *client.RancherClient
	broadcaster   *Broadcaster
}

func NewRancherServiceDriver(rancherHost string, rancherAccessKey string, rancherSecretKey string) (*RancherServiceDriver, error) {

	rancherClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       rancherHost,
		AccessKey: rancherAccessKey,
		SecretKey: rancherSecretKey,
	})

	if err != nil {
		return nil, err
	}

	sd := &RancherServiceDriver{
		rancherClient,
		NewBroadcaster(),
	}

	c, _, err := getRancherSocket(rancherClient)
	go sd.watch(c)

	return sd, nil

}

func getRancherSocket(r *client.RancherClient) (*websocket.Conn, *http.Response, error) {
	rancherUrl, _ := url.Parse(r.Opts.Url)

	//TODO extract projectId from rancherHost
	u := url.URL{
		Scheme:   "ws",
		Host:     rancherUrl.Host,
		Path:     "/v1/subscribe",
		RawQuery: "eventNames=resource.change&include=hosts&include=instances&include=instance&include=instanceLinks&include=ipAddresses&projectId=1a5",
	}

	header := http.Header{
		"Authorization": []string{"Basic " + basicAuth(r.Opts.AccessKey, r.Opts.SecretKey)},
	}

	return r.Websocket(u.String(), header)
}

func (r *RancherServiceDriver) watch(c *websocket.Conn) {

	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			//TODO find a way to recover or to gracefully exit
			return
		}

		publish := &client.Publish{}
		json.Unmarshal([]byte(message), publish)
		if publish.Name == "resource.change" {

			switch publish.ResourceType {
			case "environment":
				var result client.Environment
				err := mapstructure.Decode(publish.Data["resource"], &result)
				if err != nil {
					log.Printf(err.Error())
				} else {
					info := &RancherInfoType {
						EnvironmentId: publish.ResourceId,
						EnvironmentName: result.Name,
						Location:        &Location{Host:fmt.Sprintf("lb.%s",result.Name), Port:80},
						CurrentStatus: convertRancherHealthToStatus(result.HealthState),
					}

					r.broadcaster.Write(NewModelEvent("update", info))

				}
				break
			}
		}
	}

}

func convertRancherHealthToStatus(health string) string {
	switch health {
	case "healthy":
		return STARTED_STATUS
	case "degraded","activating","initializing":
		return STARTING_STATUS
	default:
		return STOPPED_STATUS
	}
	return STOPPED_STATUS
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

	return &RancherInfoType{EnvironmentId: env.Id}, nil

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
	rancherId := s.Config.RancherInfo.EnvironmentId
	env, err := r.rancherClient.Environment.ById(rancherId)
	check(err)
	env, err = r.rancherClient.Environment.ActionActivateservices(env)
	return s, err
}

func (r *RancherServiceDriver) Stop(s *Service) (interface{}, error) {
	rancherId := s.Config.RancherInfo.EnvironmentId
	env, err := r.rancherClient.Environment.ById(rancherId)
	check(err)
	env, err = r.rancherClient.Environment.ActionDeactivateservices(env)
	return s, err
}

func (r *RancherServiceDriver) Destroy(s *Service) error {
	rancherId := s.Config.RancherInfo.EnvironmentId
	env, err := r.rancherClient.Environment.ById(rancherId)
	check(err)
	return r.rancherClient.Environment.Delete(env)
}

func check(e error) {
	if e != nil {
		log := logrus.New()
		log.Info("error: ", e.Error())
		panic(e)
	}
}

func (r *RancherServiceDriver) Listen() chan *ModelEvent {
	return FromInterfaceChannel(r.broadcaster.Listen())
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

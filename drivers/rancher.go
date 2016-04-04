package drivers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	. "github.com/arkenio/goarken/model"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/go-rancher/client"
	catalogclient "github.com/dmetzler/go-ranchercatalog/client"
	"net/http"
	"net/url"
	"regexp"
)

var (
	log               = logrus.New()
	rancherHostRegexp = regexp.MustCompile("http.*/projects/(.*)")
)

type RancherServiceDriver struct {
	rancherClient *client.RancherClient
	broadcaster   *Broadcaster
	rancherCatClient *catalogclient.RancherCatalogClient
}

func NewRancherServiceDriver(rancherHost string, rancherAccessKey string, rancherSecretKey string) (*RancherServiceDriver, error) {

	clientOpts := &client.ClientOpts{
		Url:       rancherHost,
		AccessKey: rancherAccessKey,
		SecretKey: rancherSecretKey,
	}

	catalaogClientOpts := & catalogclient.ClientOpts{
		Url: clientOpts.Url,
		AccessKey: clientOpts.AccessKey,
		SecretKey: clientOpts.SecretKey,
	}

	rancherClient, err := client.NewRancherClient(clientOpts)
	rancherCatClient, err := catalogclient.NewRancherCatalogClient(catalaogClientOpts)



	if err != nil {
		return nil, err
	}

	sd := &RancherServiceDriver{
		rancherClient,
		NewBroadcaster(),
		rancherCatClient,
	}


	c, _, err := getRancherSocket(rancherClient)
	if err != nil {
		return nil, err
	}
	go sd.watch(c)

	return sd, nil

}



func getProjectIdFromRancherHost(host string) string {
	matches := rancherHostRegexp.FindStringSubmatch(host)
	if len(matches) > 1 {
		return matches[1]
	} else {
		return ""
	}
}

func getRancherSocket(r *client.RancherClient) (*websocket.Conn, *http.Response, error) {
	rancherUrl, _ := url.Parse(r.Opts.Url)

	projectId := getProjectIdFromRancherHost(r.Opts.Url)
	u := url.URL{
		Scheme:   "ws",
		Host:     rancherUrl.Host,
		Path:     "/v1/subscribe",
		RawQuery: fmt.Sprintf("eventNames=resource.change&include=hosts&include=instances&include=instance&include=instanceLinks&include=ipAddresses&projectId=%s", projectId),
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
					//TODO : detect wich service of the stack must be proxied by gogeta by iterating over containers labels
					info := &RancherInfoType{
						EnvironmentId:   publish.ResourceId,
						EnvironmentName: result.Name,
						Location:        &Location{Host: fmt.Sprintf("lb.%s", result.Name), Port: 80},
						HealthState:     result.HealthState,
						CurrentStatus:   convertRancherHealthToStatus(result.HealthState),
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
	case "degraded", "activating", "initializing":
		return STARTING_STATUS
	default:
		return STOPPED_STATUS
	}
	return STOPPED_STATUS
}

func (r *RancherServiceDriver) Create(s *Service, startOnCreate bool) (interface{}, error) {

	info := s.Config.RancherInfo

	if info.TemplateId == "" {
		return nil, errors.New("Rancher template has to be specified !")
	}

	template,err := r.rancherCatClient.TemplateVersion.ById(info.TemplateId)
	if err != nil {
		return nil, errors.New("Rancher template not found : " + err.Error())
	}

	//Start rancher environment
	env := &client.Environment{}
	env.StartOnCreate = startOnCreate
	env.Name = s.Name
	env.Environment = s.Config.Environment
	env.DockerCompose = extractFileContent(template,"docker-compose.yml")
	env.RancherCompose = extractFileContent(template,"rancher-compose.yml")

	env, err = r.rancherClient.Environment.Create(env)

	if err != nil {
		return nil, errors.New("Error when creating service on Rancher side: " + err.Error())
	}


	return &RancherInfoType{EnvironmentId: env.Id}, nil

}

func extractFileContent(template *catalogclient.TemplateVersion, filename string) string {
	if content,ok := template.Files[filename].(string); ok {
		return content
	}
	return ""
}


func (r *RancherServiceDriver) Start(s *Service) (interface{}, error) {
	rancherId := s.Config.RancherInfo.EnvironmentId
	env, err := r.rancherClient.Environment.ById(rancherId)
	if err != nil {
		return nil, err
	}
	env, err = r.rancherClient.Environment.ActionActivateservices(env)
	return s, err
}

func (r *RancherServiceDriver) Stop(s *Service) (interface{}, error) {
	rancherId := s.Config.RancherInfo.EnvironmentId
	env, err := r.rancherClient.Environment.ById(rancherId)
	if err != nil {
		return nil, err
	}
	env, err = r.rancherClient.Environment.ActionDeactivateservices(env)
	return s, err
}

func (r *RancherServiceDriver) Destroy(s *Service) error {
	rancherId := s.Config.RancherInfo.EnvironmentId
	env, err := r.rancherClient.Environment.ById(rancherId)
	if err != nil {
		return err
	}
	return r.rancherClient.Environment.Delete(env)
}


func (r *RancherServiceDriver) Listen() chan *ModelEvent {
	return FromInterfaceChannel(r.broadcaster.Listen())
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

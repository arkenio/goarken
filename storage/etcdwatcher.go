package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/arkenio/goarken/model"
	. "github.com/arkenio/goarken/model"
	"github.com/coreos/go-etcd/etcd"
	"regexp"
	"strconv"
	"strings"
	"time"

//	"github.com/Sirupsen/logrus"
	"github.com/Sirupsen/logrus"
)

const (
	TIME_FORMAT = "2006-01-02 15:04:05"
)

var (
	domainRegexp  = regexp.MustCompile("/domain/(.*)(/.*)*")
	serviceRegexp = regexp.MustCompile("/services/(.*)(/.*)*")
	log = logrus.New()
)

// A Watcher loads and watch the etcd hierarchy for Domains and Services and
// updates the model according to etcd updates.
type Watcher struct {
	client        *etcd.Client
	broadcaster   *Broadcaster
	servicePrefix string
	domainPrefix  string
}

func NewWatcher(client *etcd.Client, servicePrefix string, domainPrefix string) *Watcher {

	watcher := &Watcher{
		client:        client,
		broadcaster:   NewBroadcaster(),
		servicePrefix: servicePrefix,
		domainPrefix:  domainPrefix,
	}

	watcher.Init()

	return watcher
}

//Init Domains and Services.
func (w *Watcher) Init() {

	setDomainPrefix(w.domainPrefix)
	SetServicePrefix(w.servicePrefix)

	if w.domainPrefix != "" {
		go w.doWatch(w.domainPrefix, w.registerDomain)
	}
	if w.servicePrefix != "" {
		go w.doWatch(w.servicePrefix, w.registerService)
	}

}

func (w *Watcher) Listen() chan *model.ModelEvent {

	return FromInterfaceChannel(w.broadcaster.Listen())


}

// Loads and watch an etcd directory to register objects like Domains, Services
// etc... The register function is passed the etcd Node that has been loaded.
func (w *Watcher) doWatch(etcdDir string, registerFunc func(*etcd.Node, string)) {
	stop := make(chan struct{})

	for {
		log.Infof("Start watching %s", etcdDir)

		updateChannel := make(chan *etcd.Response, 10)
		go w.watch(updateChannel, stop, etcdDir, registerFunc)

		_, err := w.client.Watch(etcdDir, (uint64)(0), true, updateChannel, nil)

		//If we are here, this means etcd watch ended in an error
		stop <- struct{}{}
		w.client.CloseCURL()
		log.Warningf("Error when watching %s : %v", etcdDir, err)
		log.Warningf("Waiting 1 second and relaunch watch")
		time.Sleep(time.Second)

	}

}

func (w *Watcher) loadPrefix(etcDir string, registerFunc func(*etcd.Node, string)) {
	response, err := w.client.Get(etcDir, true, true)
	if err == nil {
		for _, serviceNode := range response.Node.Nodes {
			registerFunc(serviceNode, response.Action)

		}
	}
}

func (w *Watcher) watch(updateChannel chan *etcd.Response, stop chan struct{}, key string, registerFunc func(*etcd.Node, string)) {
	for {
		select {
		case <-stop:
			log.Warningf("Gracefully closing the etcd watch for %s", key)
			return
		case response := <-updateChannel:
			if response != nil {
				registerFunc(response.Node, response.Action)
			}
		}
	}
}

//func GetDomainFromPath(domainPath string, client *etcd.Client) (*Domain, error) {
//	// Get service's root node instead of changed node.
//	response, err := client.Get(domainPath, true, true)
//	if err != nil {
//		return nil, errors.New(fmt.Sprintf("Unable to get information for service %s from etcd", domainPath))
//	}
//
//	return GetDomainFromNode(response.Node), nil
//}

func GetDomainFromNode(node *etcd.Node) (*Domain,error) {
	return NewDomain(node)
}

func GetServiceClusterFromPath(serviceClusterPath string, client *etcd.Client) (*ServiceCluster, error) {
	// Get service's root node instead of changed node.
	response, err := client.Get(serviceClusterPath, true, true)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to get information for service %s from etcd", serviceClusterPath))
	}

	return getServiceClusterFromNode(response.Node), nil
}

func getServiceClusterFromNode(clusterNode *etcd.Node) *ServiceCluster {

	sc := NewServiceCluster(clusterNode.Key)
	for _, indexNode := range clusterNode.Nodes {
		service, err := newService(indexNode)
		if err == nil {
			sc.Add(service)
			sc.Name = service.Name
		}
	}

	return sc

}

func (w *Watcher) registerDomain(node *etcd.Node, action string) {

	domainName, err := getDomainForNode(node)
	if err != nil {
		return
	}

	if action == "delete" || action == "expire" {
		w.broadcaster.Write(NewModelEvent("delete", &Domain{Name: domainName}))
		return
	}

	domainKey := w.domainPrefix + "/" + domainName
	response, err := w.client.Get(domainKey, true, false)

	if err == nil {
		domain, _ := NewDomain(response.Node)

		//actualDomain := w.model.Domains[domainName]

		if domain.Typ != "" && domain.Value != "" { // && !domain.Equals(actualDomain) {
			//w.model.Domains[domainName] = domain
			log.Infof("Registered domain %s with (%s) %s", domainName, domain.Typ, domain.Value)

			//Broadcast the updated domain
			w.broadcaster.Write(model.NewModelEvent("update", domain))

		}

	}

}

func NewDomain(domainNode *etcd.Node) (*Domain, error) {
	domain := &Domain{}

	domain.NodeKey = domainNode.Key

	domainName, err := getDomainForNode(domainNode)
	if err != nil {
		return nil, err
	}
	domain.Name = domainName

	for _, node := range domainNode.Nodes {
		switch node.Key {
		case domainNode.Key + "/type":
			domain.Typ = node.Value
		case domainNode.Key + "/value":
			domain.Value = node.Value
		}
	}
	return domain,nil

}

func (w *Watcher) registerService(node *etcd.Node, action string) {

	serviceName, err := getEnvForNode(node)
	if err != nil {
		return
	}

	if action == "delete" && node.Key == w.servicePrefix+"/"+serviceName {
		w.broadcaster.Write(NewModelEvent("delete", &ServiceCluster{Name: serviceName}))
		return
	}

	// Get service's root node instead of changed node.
	response, err := w.client.Get(w.servicePrefix+"/"+serviceName, true, true)

	if err == nil {

		sc := getServiceClusterFromNode(response.Node)

		w.broadcaster.Write(model.NewModelEvent("update", sc))

		/*if w.model.Services[sc.Name] == nil {
			w.model.Services[sc.Name] = sc
			w.broadcaster.Write(w.model.Services[serviceName])

		} else {
			for _, service := range sc.GetInstances() {
				actualEnv := w.model.Services[serviceName].Get(service.Index)
				if !actualEnv.Equals(service) {
					w.model.Services[serviceName].Add(service)
					if service.Location.Host != "" && service.Location.Port != 0 {
						log.Infof("Registering service %s with location : http://%s:%d/", serviceName, service.Location.Host, service.Location.Port)
					} else {
						log.Infof("Registering service %s without location", serviceName)
					}
					//Broadcast the updated object
				w.broadcaster.Write(w.model.Services[serviceName])
				}
			}
		}*/

	} else {
		log.Errorf("Unable to get information for service %s from etcd", serviceName)
	}
}

func newService(serviceNode *etcd.Node) (*Service, error) {

	serviceIndex, err := getEnvIndexForNode(serviceNode)
	if err != nil {
		return nil, err
	}

	serviceName, err := getEnvForNode(serviceNode)
	if err != nil {
		return nil, err
	}

	if _, err := strconv.Atoi(serviceIndex); err != nil {
		// Don't handle node that are not integer (ie config node)
		return nil, errors.New("Not a service index node")
	}

	service := &Service{}
	service.Location = &Location{}
	service.Config = &ServiceConfig{Robots: ""}
	service.Index = serviceIndex
	service.Name = serviceName
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
				log.Errorf("Error parsing last access date with service %s: %s", service.Name, err)
				break
			}
			service.LastAccess = &lastAccessTime

		case service.NodeKey + "/status":
			service.Status = NewStatus(service, node)
		}
	}
	return service, nil
}

func (w *Watcher) PersistService(s *Service) (*Service, error) {
	if s.NodeKey != "" {
		log.Debugf("Persisting key %s ", s.NodeKey)
		resp, err := w.client.Get(s.NodeKey, false, true)
		if(err != nil) {
			return nil, errors.New("No service with key " + s.NodeKey + " found in etcd")
		}

		oldService, err := newService(resp.Node)
		if err != nil {
			return nil, err
		} else {
			if oldService.Status.Expected != s.Status.Expected {
				_,err = w.client.Set(fmt.Sprintf("%s/status/expected", s.NodeKey), s.Status.Expected, 0)
			}

			if err == nil &&  oldService.Status.Current != s.Status.Current {
				_,err =  w.client.Set(fmt.Sprintf("%s/status/current", s.NodeKey), s.Status.Current, 0)
			}

			if err == nil &&  oldService.Status.Alive != s.Status.Alive {
				_,err =  w.client.Set(fmt.Sprintf("%s/status/alive", s.NodeKey), s.Status.Alive, 0)
			}


			bytes, err2 := json.Marshal(s.Location)
			if err2 == nil &&  oldService.Location != s.Location{
				_,err =  w.client.Set(fmt.Sprintf("%s/location", s.NodeKey), string(bytes), 0)
			} else {
				err = err2
			}


			bytes, err2 = json.Marshal(s.Config)
			if err2 == nil {
				_, err = w.client.Set(fmt.Sprintf("%s/config/gogeta", s.NodeKey), string(bytes), 0)
			}else {
				err = err2
			}

			if err == nil && oldService.Domain != s.Domain {
				_,err = w.client.Set(fmt.Sprintf("%s/domain", s.NodeKey), s.Domain, 0)
			}

			if(err != nil ) {
				return nil,err
			}

		}

	} else {
		s.NodeKey = computeNodeKey(s, w.servicePrefix)

		_, err := w.client.Create(fmt.Sprintf("%s/status/expected", s.NodeKey), s.Status.Expected, 0)
		if err == nil {
			_, err = w.client.Create(fmt.Sprintf("%s/status/current", s.NodeKey), s.Status.Current, 0)
		}
		if err == nil {
			bytes, err := json.Marshal(s.Config)
			if err == nil {
				_, err = w.client.Create(fmt.Sprintf("%s/config/gogeta", s.NodeKey), string(bytes), 0)
			}
		}
		if err == nil {
			_, err = w.client.Create(fmt.Sprintf("%s/domain", s.NodeKey), s.Domain, 0)
		}

		if err != nil {
			//Rollback creation
			w.client.Delete(s.NodeKey, true)
			return nil, err
		}

	}
	return s, nil

}

func computeNodeKey(s *Service, servicePrefix string) string {
	return fmt.Sprintf("/%s/%s/%s", servicePrefix, s.Name, s.Index)
}

func computeDomainNodeKey(domainName string, domainPrefix string) string {
	return fmt.Sprintf("/%s/%s", domainPrefix, domainName)
}

func computeClusterKey(serviceName string, servicePrefix string) string {
	return fmt.Sprintf("/%s/%s/", servicePrefix, serviceName)
}

func getEnvIndexForNode(node *etcd.Node) (string, error) {
	matches := serviceRegexp.FindStringSubmatch(node.Key)
	if len(matches) > 1 {
		parts := strings.Split(matches[1], "/")
		return parts[1], nil
	} else {
		return "", errors.New("Unable to extract env for node " + node.Key)
	}
}

func getEnvForNode(node *etcd.Node) (string, error) {
	matches := serviceRegexp.FindStringSubmatch(node.Key)
	if len(matches) > 1 {
		parts := strings.Split(matches[1], "/")
		return parts[0], nil
	} else {
		return "", errors.New("Unable to extract env for node " + node.Key)
	}
}

func setDomainPrefix(domainPrefix string) {
	domainRegexp = regexp.MustCompile(domainPrefix + "/(.*)(/.*)*")
}

func getDomainForNode(node *etcd.Node) (string, error) {
	matches := domainRegexp.FindStringSubmatch(node.Key)
	if len(matches) > 1 {
		parts := strings.Split(matches[1], "/")
		return parts[0], nil
	} else {
		return "", errors.New("Unable to extract domain for node " + node.Key)
	}
}

func SetServicePrefix(servicePrefix string) {
	serviceRegexp = regexp.MustCompile(servicePrefix + "/(.*)(/.*)*")
}

func NewStatus(service *Service, node *etcd.Node) *Status {
	status := &Status{}
	statusKey := service.NodeKey + "/status"
	status.Service = service
	for _, subNode := range node.Nodes {
		switch subNode.Key {
		case statusKey + "/alive":
			status.Alive = subNode.Value
		case statusKey + "/current":
			status.Current = subNode.Value
		case statusKey + "/expected":
			status.Expected = subNode.Value
		}
	}
	return status
}

func (w *Watcher) LoadAllServices() (map[string]*ServiceCluster, error) {
	result := make(map[string]*ServiceCluster)

	response, err := w.client.Get(w.servicePrefix, true, true)
	if err == nil {
		for _, serviceNode := range response.Node.Nodes {
			sc := getServiceClusterFromNode(serviceNode)
			result[sc.Name] = sc
		}
	} else {
		return nil, err
	}
	return result, nil
}

func (w *Watcher) LoadService(serviceName string) (*ServiceCluster, error) {

	response, err := w.client.Get(computeClusterKey(serviceName, w.servicePrefix), true, true)
	if err != nil {
		return nil, err
	} else {
		return getServiceClusterFromNode(response.Node), nil
	}

}

func (w *Watcher) DestroyService(sc *ServiceCluster) error {
	_, err := w.client.Delete(computeClusterKey(sc.Name, w.servicePrefix), true)

	return err
}

func (w *Watcher) LoadAllDomains() (map[string]*Domain, error) {

	result := make(map[string]*Domain)

	response, err := w.client.Get(w.domainPrefix, true, true)
	if err == nil {
		for _, domainNode := range response.Node.Nodes {
			domain, err := NewDomain(domainNode)
			if err == nil {
				result[domain.Name] = domain
			}
		}
	} else {
		return nil, err
	}
	return result, nil
}
func (w *Watcher) LoadDomain(domainName string) (*Domain, error) {
	response, err := w.client.Get(computeDomainNodeKey(domainName, w.domainPrefix), true, true)
	if err != nil {
		return nil, err
	} else {
		domain, _ :=NewDomain(response.Node)
		return domain, nil
	}

}

func (w *Watcher) PersistDomain(d *Domain) (*Domain, error) {

	if d.NodeKey != "" {
		resp, err := w.client.Get(d.NodeKey, false, true)

		oldDomain, _ := NewDomain(resp.Node)
		if err != nil {
			return nil, err
		} else {
			if oldDomain.Typ != d.Typ {
				w.client.Set(fmt.Sprintf("%s/type", d.NodeKey), d.Typ, 0)
			}

			if oldDomain.Value != d.Value {
				w.client.Set(fmt.Sprintf("%s/value", d.NodeKey), d.Value, 0)
			}
		}

	} else {

		d.NodeKey = computeDomainNodeKey(d.Name, w.domainPrefix)
 		_, err := w.client.Create(fmt.Sprintf("%s/type", d.NodeKey), d.Typ, 0)
		if err == nil {
			_, err = w.client.Create(fmt.Sprintf("%s/value", d.NodeKey), d.Value, 0)
		}

		if err != nil {
			//Rollback creation
			w.client.Delete(d.NodeKey, true)
			return nil, err
		}

	}
	return d, nil

}
func (w *Watcher) DestroyDomain(d *Domain) error {
	_, err := w.client.Delete(d.NodeKey, true)
	return err

}

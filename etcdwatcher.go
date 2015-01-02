package goarken

import (
	"errors"
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
	"time"
)

const (
	TIME_FORMAT = "2006-01-02 15:04:05"
)

// A Watcher loads and watch the etcd hierarchy for Domains and Services.
type Watcher struct {
	Client        *etcd.Client
	DomainPrefix  string
	ServicePrefix string
	Domains       map[string]*Domain
	Services      map[string]*ServiceCluster
	broadcaster   *Broadcaster
}

//Init Domains and Services.
func (w *Watcher) Init() {
	w.broadcaster = NewBroadcaster()
	SetServicePrefix(w.ServicePrefix)
	SetDomainPrefix(w.DomainPrefix)
	if w.Domains != nil {
		w.loadPrefix(w.DomainPrefix, w.registerDomain)
		go w.doWatch(w.DomainPrefix, w.registerDomain)
	}
	if w.Services != nil {
		w.loadPrefix(w.ServicePrefix, w.registerService)
		go w.doWatch(w.ServicePrefix, w.registerService)
	}

}

func (w *Watcher) Listen() chan interface{} {
	return w.broadcaster.Listen()
}

// Loads and watch an etcd directory to register objects like Domains, Services
// etc... The register function is passed the etcd Node that has been loaded.
func (w *Watcher) doWatch(etcdDir string, registerFunc func(*etcd.Node, string)) {
	stop := make(chan struct{})

	for {
		glog.Infof("Start watching %s", etcdDir)

		updateChannel := make(chan *etcd.Response, 10)
		go w.watch(updateChannel, stop, etcdDir, registerFunc)

		_, err := w.Client.Watch(etcdDir, (uint64)(0), true, updateChannel, nil)

		//If we are here, this means etcd watch ended in an error
		stop <- struct{}{}
		w.Client.CloseCURL()
		glog.Warningf("Error when watching %s : %v", etcdDir, err)
		glog.Warningf("Waiting 1 second and relaunch watch")
		time.Sleep(time.Second)

	}

}

func (w *Watcher) loadPrefix(etcDir string, registerFunc func(*etcd.Node, string)) {
	response, err := w.Client.Get(etcDir, true, true)
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
			glog.Warningf("Gracefully closing the etcd watch for %s", key)
			return
		case response := <-updateChannel:
			if response != nil {
				registerFunc(response.Node, response.Action)
			}
		default:
			// Don't slam the etcd server
			time.Sleep(time.Second)
		}
	}
}

func (w *Watcher) RemoveDomain(key string) {
	delete(w.Domains, key)

}

func (w *Watcher) RemoveEnv(serviceName string) {
	delete(w.Services, serviceName)
}

func GetDomainFromPath(domainPath string, client *etcd.Client) (*Domain, error) {
	// Get service's root node instead of changed node.
	response, err := client.Get(domainPath, true, true)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to get information for service %s from etcd", domainPath))
	}

	return GetDomainFromNode(response.Node), nil
}

func GetDomainFromNode(node *etcd.Node) *Domain {
	return NewDomain(node)
}

func GetServiceClusterFromPath(serviceClusterPath string, client *etcd.Client) (*ServiceCluster, error) {
	// Get service's root node instead of changed node.
	response, err := client.Get(serviceClusterPath, true, true)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to get information for service %s from etcd", serviceClusterPath))
	}

	return GetServiceClusterFromNode(response.Node), nil
}

func GetServiceClusterFromNode(clusterNode *etcd.Node) *ServiceCluster {

	sc := NewServiceCluster("")
	for _, indexNode := range clusterNode.Nodes {
		service, err := NewService(indexNode)
		if err == nil {
			sc.Add(service)
			sc.Name = service.Name
		}
	}
	return sc

}

func (w *Watcher) registerDomain(node *etcd.Node, action string) {

	domainName := getDomainForNode(node)

	if action == "delete" || action == "expire" {
		w.RemoveDomain(domainName)
		return
	}

	domainKey := w.DomainPrefix + "/" + domainName
	response, err := w.Client.Get(domainKey, true, false)

	if err == nil {
		domain := NewDomain(response.Node)

		actualDomain := w.Domains[domainName]

		if domain.Typ != "" && domain.Value != "" && !domain.Equals(actualDomain) {
			w.Domains[domainName] = domain
			glog.Infof("Registered domain %s with (%s) %s", domainName, domain.Typ, domain.Value)

			//Broadcast the updated domain
			w.broadcaster.Write(domain)

		}

	}

}

func (w *Watcher) registerService(node *etcd.Node, action string) {

	serviceName := getEnvForNode(node)

	if action == "delete" && node.Key == w.ServicePrefix+"/"+serviceName {
		w.RemoveEnv(serviceName)
		return
	}

	// Get service's root node instead of changed node.
	response, err := w.Client.Get(w.ServicePrefix+"/"+serviceName, true, true)

	if err == nil {

		sc := GetServiceClusterFromNode(response.Node)

		if w.Services[sc.Name] == nil {
			w.Services[sc.Name] = sc
			w.broadcaster.Write(w.Services[serviceName])

		} else {
			for _, service := range sc.GetInstances() {
				actualEnv := w.Services[serviceName].Get(service.Index)
				if !actualEnv.Equals(service) {
					w.Services[serviceName].Add(service)
					if service.Location.Host != "" && service.Location.Port != 0 {
						glog.Infof("Registering service %s with location : http://%s:%d/", serviceName, service.Location.Host, service.Location.Port)
					} else {
						glog.Infof("Registering service %s without location", serviceName)
					}
					//Broadcast the updated object
					w.broadcaster.Write(w.Services[serviceName])
				}
			}
		}

	} else {
		glog.Errorf("Unable to get information for service %s from etcd", serviceName)
	}
}

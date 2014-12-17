package goarken

import (
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
	"regexp"
	"strings"
	"time"
)

// A Watcher loads and watch the etcd hierarchy for Domains and Services.
type Watcher struct {
	Client        *etcd.Client
	DomainPrefix  string
	ServicePrefix string
	Domains       map[string]*Domain
	Services      map[string]*ServiceCluster
}

//Init Domains and Services.
func (w *Watcher) Init() {
	if w.Domains != nil {
		go w.loadAndWatch(w.DomainPrefix, w.registerDomain)
	}
	if w.Services != nil {
		go w.loadAndWatch(w.ServicePrefix, w.registerService)
	}

}

// Loads and watch an etcd directory to register objects like Domains, Services
// etc... The register function is passed the etcd Node that has been loaded.
func (w *Watcher) loadAndWatch(etcdDir string, registerFunc func(*etcd.Node, string)) {
	w.loadPrefix(etcdDir, registerFunc)
	stop := make(chan struct{})

	for {
		glog.Infof("Start watching %s", etcdDir)

		updateChannel := make(chan *etcd.Response, 10)
		go w.watch(updateChannel, stop, etcdDir, registerFunc)

		_, err := w.Client.Watch(etcdDir, (uint64)(0), true, updateChannel, nil)

		//If we are here, this means etcd watch ended in an error
		stop <- struct{}{}
		w.Client.CloseCURL()
		glog.Errorf("Error when watching %s : %v", etcdDir, err)
		glog.Errorf("Waiting 1 second and relaunch watch")
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

func (w *Watcher) registerDomain(node *etcd.Node, action string) {

	domainName := w.getDomainForNode(node)

	domainKey := w.DomainPrefix + "/" + domainName
	response, err := w.Client.Get(domainKey, true, false)

	if action == "delete" || action == "expire" {
		w.RemoveDomain(domainName)
		return
	}

	if err == nil {
		domain := &Domain{}
		for _, node := range response.Node.Nodes {
			switch node.Key {
			case domainKey + "/type":
				domain.typ = node.Value
			case domainKey + "/value":
				domain.value = node.Value
			}
		}

		actualDomain := w.Domains[domainName]

		if domain.typ != "" && domain.value != "" && !domain.Equals(actualDomain) {
			w.Domains[domainName] = domain
			glog.Infof("Registered domain %s with (%s) %s", domainName, domain.typ, domain.value)

		}
	}

}

func (w *Watcher) RemoveDomain(key string) {
	delete(w.Domains, key)

}

func (w *Watcher) getDomainForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.DomainPrefix + "/(.*)")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (w *Watcher) getEnvForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.ServicePrefix + "/(.*)(/.*)*")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (w *Watcher) getEnvIndexForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.ServicePrefix + "/(.*)(/.*)*")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[1]
}

func (w *Watcher) RemoveEnv(serviceName string) {
	delete(w.Services, serviceName)
}

func (w *Watcher) registerService(node *etcd.Node, action string) {
	serviceName := w.getEnvForNode(node)

	// Get service's root node instead of changed node.
	serviceNode, err := w.Client.Get(w.ServicePrefix+"/"+serviceName, true, true)

	if err == nil {

		for _, indexNode := range serviceNode.Node.Nodes {

			serviceIndex := w.getEnvIndexForNode(indexNode)
			serviceKey := w.ServicePrefix + "/" + serviceName + "/" + serviceIndex
			statusKey := serviceKey + "/status"

			response, err := w.Client.Get(serviceKey, true, true)

			if err == nil {

				if w.Services[serviceName] == nil {
					w.Services[serviceName] = &ServiceCluster{}
				}

				service := &Service{}
				service.location = &location{}
				service.index = serviceIndex
				service.nodeKey = serviceKey
				service.name = serviceName

				if action == "delete" {
					glog.Infof("Removing service %s", serviceName)
					w.RemoveEnv(serviceName)
					return
				}

				for _, node := range response.Node.Nodes {
					switch node.Key {
					case serviceKey + "/location":
						location := &location{}
						err := json.Unmarshal([]byte(node.Value), location)
						if err == nil {
							service.location.Host = location.Host
							service.location.Port = location.Port
						}

					case serviceKey + "/domain":
						service.domain = node.Value

					case statusKey:
						service.status = &Status{}
						service.status.service = service
						for _, subNode := range node.Nodes {
							switch subNode.Key {
							case statusKey + "/alive":
								service.status.alive = subNode.Value
							case statusKey + "/current":
								service.status.current = subNode.Value
							case statusKey + "/expected":
								service.status.expected = subNode.Value
							}
						}
					}
				}

				actualEnv := w.Services[serviceName].Get(service.index)

				if !actualEnv.Equals(service) {
					w.Services[serviceName].Add(service)
					if service.location.Host != "" && service.location.Port != 0 {
						glog.Infof("Registering service %s with location : http://%s:%d/", serviceName, service.location.Host, service.location.Port)
					} else {
						glog.Infof("Registering service %s without location", serviceName)
					}

				}
			}
		}
	} else {
		glog.Errorf("Unable to get information for service %s from etcd", serviceName)
	}
}

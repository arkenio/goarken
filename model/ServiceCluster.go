package model

import (
	"errors"
	"sync"
)

type ServiceCluster struct {
	Name      string     `json:"name"`
	Instances []*Service `json:"instances"`
	lastIndex int
	lock      sync.RWMutex
}

func NewServiceCluster(name string) *ServiceCluster {
	sc := &ServiceCluster{
		Name: name,
	}
	return sc
}

func (cl *ServiceCluster) Next() (*Service, error) {
	if cl == nil {
		return nil, StatusError{}
	}
	cl.lock.RLock()
	defer cl.lock.RUnlock()
	if len(cl.Instances) == 0 {
		return nil, errors.New("no alive instance found")
	}
	var instance *Service
	for tries := 0; tries < len(cl.Instances); tries++ {
		index := (cl.lastIndex + 1) % len(cl.Instances)
		cl.lastIndex = index

		instance = cl.Instances[index]
		log.Debugf("Checking instance %d Status : %s", index, instance.Status.Compute())
		if instance.Status.Compute() == STARTED_STATUS && instance.Location.IsFullyDefined() {
			return instance, nil
		}

	}

	lastStatus := instance.Status

	if lastStatus == nil && !instance.Location.IsFullyDefined() {
		// Generates too much garbage
		log.Debugf("No Status and no location for %s", instance.Name)
		return nil, StatusError{ERROR_STATUS, lastStatus}
	}

	log.Debugf("No instance started for %s", instance.Name)
	if lastStatus != nil {
		log.Debugf("Last Status :")
		log.Debugf("   current  : %s", lastStatus.Current)
		log.Debugf("   expected : %s", lastStatus.Expected)
		log.Debugf("   alive : %s", lastStatus.Alive)
	} else {
		log.Debugf("No status available")
	}

	return nil, StatusError{instance.Status.Compute(), lastStatus}
}

func (cl *ServiceCluster) Remove(instanceIndex string) {

	match := -1
	for k, v := range cl.Instances {
		if v.Index == instanceIndex {
			match = k
		}
	}

	cl.Instances = append(cl.Instances[:match], cl.Instances[match+1:]...)
	cl.Dump("remove")
}

// Get an service by its key (index). Returns nil if not found.
func (cl *ServiceCluster) Get(instanceIndex string) *Service {
	for i, v := range cl.Instances {
		if v.Index == instanceIndex {
			return cl.Instances[i]
		}
	}
	return nil
}

func (cl *ServiceCluster) Add(service *Service) {

	for index, v := range cl.Instances {
		if v.Index == service.Index {
			cl.Instances[index] = service
			return
		}
	}

	cl.Instances = append(cl.Instances, service)
}

func (cl *ServiceCluster) Dump(action string) {
	for _, v := range cl.Instances {
		log.Debugf("Dump after %s %s -> %s:%d", action, v.Index, v.Location.Host, v.Location.Port)
	}
}

func (cl *ServiceCluster) GetInstances() []*Service {
	return cl.Instances
}

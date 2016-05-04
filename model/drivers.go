// Copyright © 2016 Nuxeo SA (http://nuxeo.com/) and others.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package model

type ServiceDriver interface {
	Create(s *Service, startOnCreate bool) (interface{}, error)
	Start(s *Service) (interface{}, error)
	Stop(s *Service) (interface{}, error)
	Destroy(s *Service) error

	Listen() chan *ModelEvent
}

type PersistenceDriver interface {
	LoadAllServices() (map[string]*ServiceCluster, error)
	LoadService(serviceName string) (*ServiceCluster, error)
	PersistService(*Service) (*Service, error)
	DestroyService(*ServiceCluster) error

	LoadAllDomains() (map[string]*Domain, error)
	LoadDomain(serviceName string) (*Domain, error)
	PersistDomain(*Domain) (*Domain, error)
	DestroyDomain(*Domain) error

	Listen() chan *ModelEvent
}

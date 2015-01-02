package goarken

import (
	"github.com/coreos/go-etcd/etcd"
	"regexp"
	"strings"
)

type Domain struct {
	Typ   string
	Value string
}

var (
	domainRegexp = regexp.MustCompile("/domain/(.*)(/.*)*")
)

func NewDomain(domainNode *etcd.Node) *Domain {
	domain := &Domain{}
	domainKey := domainNode.Key
	for _, node := range domainNode.Nodes {
		switch node.Key {
		case domainKey + "/type":
			domain.Typ = node.Value
		case domainKey + "/value":
			domain.Value = node.Value
		}
	}
	return domain

}

func setDomainPrefix(domainPrefix string) {
	domainRegexp = regexp.MustCompile(domainPrefix + "/(.*)(/.*)*")
}

func getDomainForNode(node *etcd.Node) string {
	return strings.Split(domainRegexp.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (domain *Domain) Equals(other *Domain) bool {
	if domain == nil && other == nil {
		return true
	}

	return domain != nil && other != nil &&
		domain.Typ == other.Typ && domain.Value == other.Value
}

package goarken

import "github.com/coreos/go-etcd/etcd"

const (
	STARTING_STATUS   = "starting"
	STARTED_STATUS    = "started"
	STOPPING_STATUS   = "stopping"
	STOPPED_STATUS    = "stopped"
	ERROR_STATUS      = "error"
	WARNING_STATUS	  = "warning"
	NA_STATUS         = "n/a"
	PASSIVATED_STATUS = "passivated"
)

type Status struct {
	Alive    string
	Current  string
	Expected string
	Service  *Service
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

type StatusError struct {
	ComputedStatus string
	Status         *Status
}

func (s StatusError) Error() string {
	return s.ComputedStatus
}

func (s *Status) Equals(other *Status) bool {
	if s == nil && other == nil {
		return true
	}
	return s != nil && other != nil && s.Alive == other.Alive &&
		s.Current == other.Current &&
		s.Expected == other.Expected
}

func (s *Status) Compute() string {

	if s != nil {
		Alive := s.Alive
		Expected := s.Expected
		Current := s.Current
		switch Current {
		case STOPPED_STATUS:
			if Expected == PASSIVATED_STATUS {
				return PASSIVATED_STATUS
			} else if Expected == STOPPED_STATUS {
				return STOPPED_STATUS
			} else {
				return ERROR_STATUS
			}
		case STARTING_STATUS:
			if Expected == STARTED_STATUS {
				return STARTING_STATUS
			} else {
				return ERROR_STATUS
			}
		case STARTED_STATUS:
			if Alive != "" {
				if Expected != STARTED_STATUS {
					return WARNING_STATUS
				}
				return STARTED_STATUS
			} else {
				if Expected != STARTED_STATUS {
					return WARNING_STATUS
				}
				return ERROR_STATUS
			}
		case STOPPING_STATUS:
			if Expected == STOPPED_STATUS {
				return STOPPED_STATUS
			} else {
				return ERROR_STATUS
			}
			// N/A
		default:
			return ERROR_STATUS
		}
	}
	return NA_STATUS
}

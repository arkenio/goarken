package goarken

const (
	STARTING_STATUS   = "starting"
	STARTED_STATUS    = "started"
	STOPPING_STATUS   = "stopping"
	STOPPED_STATUS    = "stopped"
	ERROR_STATUS      = "error"
	NA_STATUS         = "n/a"
	PASSIVATED_STATUS = "passivated"
)

type Status struct {
	alive    string
	current  string
	expected string
	service  *Service
}

type StatusError struct {
	computedStatus string
	status         *Status
}

func (s StatusError) Error() string {
	return s.computedStatus
}

func (s *Status) equals(other *Status) bool {
	if s == nil && other == nil {
		return true
	}
	return s != nil && other != nil && s.alive == other.alive &&
		s.current == other.current &&
		s.expected == other.expected
}

func (s *Status) compute() string {

	if s != nil {
		alive := s.alive
		expected := s.expected
		current := s.current
		switch current {
		case STOPPED_STATUS:
			if expected == PASSIVATED_STATUS {
				return PASSIVATED_STATUS
			} else if expected == STOPPED_STATUS {
				return STOPPED_STATUS
			} else {
				return ERROR_STATUS
			}
		case STARTING_STATUS:
			if expected == STARTED_STATUS {
				return STARTING_STATUS
			} else {
				return ERROR_STATUS
			}
		case STARTED_STATUS:
			if alive != "" {
				if expected != STARTED_STATUS {
					return ERROR_STATUS
				}
				return STARTED_STATUS
			} else {
				return ERROR_STATUS
			}
		case STOPPING_STATUS:
			if expected == STOPPED_STATUS {
				return STOPPED_STATUS
			} else {
				return ERROR_STATUS
			}
			// N/A
		default:
			return ERROR_STATUS
		}
	}
	return STARTED_STATUS
}

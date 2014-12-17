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
	Alive    string
	Current  string
	Expected string
	Service  *Service
}

type StatusError struct {
	computedStatus string
	status         *Status
}

func (s StatusError) Error() string {
	return s.computedStatus
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
					return ERROR_STATUS
				}
				return STARTED_STATUS
			} else {
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
	return STARTED_STATUS
}

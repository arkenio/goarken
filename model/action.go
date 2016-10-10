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

const (
	START_ACTION         = "start"
	STOP_ACTION          = "stop"
	DELETE_ACTION        = "delete"
	UPDATE_ACTION        = "update"
	UPGRADE_ACTION       = "upgrade"
	FINISHUPGRADE_ACTION = "finishupgrade"
	ROLLBACK_ACTION      = "rollback"
)

//compute default actions based on the status of the service
//avoids to return a list of actions when the service is in starting or stopping status
//doesn't modify the list of actions on the service
func GetDefaultActionsForStatus(s *Service) []string {
	if s.Status != nil {
		actions := make([]string, 0)
		Current := s.Status.Current
		Expected := s.Status.Expected
		switch Current {
		case STOPPED_STATUS:
			if len(s.Actions) > 0 {
				return s.Actions
			}
			if Expected == PASSIVATED_STATUS {
				actions = append(actions, START_ACTION, DELETE_ACTION, UPDATE_ACTION)
				return actions
			} else if Expected == STOPPED_STATUS {
				actions = append(actions, START_ACTION, UPDATE_ACTION)
			} else {
				return actions
			}
		case PASSIVATED_STATUS:
			if len(s.Actions) > 0 {
				return s.Actions
			}
			if Expected == PASSIVATED_STATUS {
				actions = append(actions, START_ACTION, DELETE_ACTION)
			} else {
				return actions
			}
		case STARTING_STATUS:
			return actions
		case STARTED_STATUS:
			if len(s.Actions) > 0 {
				return s.Actions
			}
			actions = append(actions, DELETE_ACTION, UPDATE_ACTION, STOP_ACTION)
		case STOPPING_STATUS:
			return actions
		default:
			return actions
		}
		return actions
	}
	return s.Actions
}

// called on create service, the service is stopped
func InitActions(s *Service) {

	if s.Actions == nil {
		s.Actions = make([]string, 0)
	}
	s.Actions = append(s.Actions, START_ACTION, DELETE_ACTION, UPDATE_ACTION)
}

func AddAction(s *Service, actions ...string) {
	if s.Actions == nil {
		s.Actions = make([]string, 0)
	}
	for _, action := range actions {
		canAdd := true
		switch action {
		case START_ACTION:
			for i, a := range s.Actions {
				if a == action {
					canAdd = false
				}
				if a == STOP_ACTION { //remove stop
					s.Actions = append(s.Actions[:i], s.Actions[i+1:]...)
				}
			}
		case STOP_ACTION:
			for i, a := range s.Actions {
				if a == action {
					canAdd = false
				}
				if a == START_ACTION {
					s.Actions = append(s.Actions[:i], s.Actions[i+1:]...)
				}
			}
		case DELETE_ACTION:
			for _, a := range s.Actions {
				if a == action {
					canAdd = false
				}
			}
		case UPDATE_ACTION:
			var actions = make([]string, 0)
			for _, a := range s.Actions {
				if a != UPGRADE_ACTION && a != FINISHUPGRADE_ACTION || a != ROLLBACK_ACTION {
					actions = append(actions, a)
				}
				if a == action {
					canAdd = false
				}
			}
			s.Actions = actions
		case UPGRADE_ACTION:
			for i, a := range s.Actions {
				if a == UPDATE_ACTION {
					s.Actions = append(s.Actions[:i], s.Actions[i+1:]...)
				}
				if a == action || a == FINISHUPGRADE_ACTION || a == ROLLBACK_ACTION {
					canAdd = false
				}

			}
		case FINISHUPGRADE_ACTION:
			for i, a := range s.Actions {
				if a == action {
					canAdd = false
				}
				if a == UPGRADE_ACTION {
					s.Actions = append(s.Actions[:i], s.Actions[i+1:]...)
				}
			}
		case ROLLBACK_ACTION:
			for i, a := range s.Actions {
				if a == action {
					canAdd = false
				}
				if a == UPGRADE_ACTION {
					s.Actions = append(s.Actions[:i], s.Actions[i+1:]...)
				}
			}
		}
		if canAdd {
			s.Actions = append(s.Actions, action)
		}
	}
}

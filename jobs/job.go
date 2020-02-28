package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// Job is the detail about a job
type Job struct {
	Description        string
	Details            map[string]string
	Enabled            bool
	ID                 string
	ModifiedBy         string
	ModifiedAt         *time.Time
	Name               string
	Group              string
	ScheduleExpression string
}

// NewID returns a new ID for a job.  Currently this is just a UUID string
func NewID() string {
	id := uuid.New().String()

	log.Debugf("generated random job id %s", id)

	return id
}

// UnmarshalJSON is a custom JSON unmarshaller for metadata
func (m *Job) UnmarshalJSON(j []byte) error {
	var rawStrings map[string]interface{}

	log.Debugf("unmarshalling job: %s", string(j))

	err := json.Unmarshal(j, &rawStrings)
	if err != nil {
		return err
	}

	log.Debug("unmarshaled metadata into rawstrings")

	if desc, ok := rawStrings["description"]; ok {
		if s, ok := desc.(string); !ok {
			msg := fmt.Sprintf("description is not a string: %+v", rawStrings["description"])
			return errors.New(msg)
		} else {
			m.Description = s
		}
	}

	if d, ok := rawStrings["details"]; ok {
		details := make(map[string]string)
		if i, ok := d.(map[string]interface{}); !ok {
			msg := fmt.Sprintf("details is not a map of strings: %+v", rawStrings["details"])
			return errors.New(msg)
		} else {
			for k, v := range i {
				if s, ok := v.(string); ok {
					details[k] = s
				} else {
					msg := fmt.Sprintf("invalid type in details map, value of '%s' is not a string %T(%v)'", k, v, v)
					return errors.New(msg)
				}
			}
		}

		m.Details = details
	}

	if group, ok := rawStrings["group"]; ok {
		if s, ok := group.(string); !ok {
			msg := fmt.Sprintf("group is not a string: %+v", rawStrings["group"])
			return errors.New(msg)
		} else {
			m.Group = s
		}
	}

	if id, ok := rawStrings["id"]; ok {
		if s, ok := id.(string); !ok {
			msg := fmt.Sprintf("id is not a string: %+v", rawStrings["id"])
			return errors.New(msg)
		} else {
			m.ID = s
		}
	}

	if modifiedAt, ok := rawStrings["modified_at"]; ok {
		if ma, ok := modifiedAt.(string); !ok {
			msg := fmt.Sprintf("modified_at is not a string: %+v", rawStrings["modified_at"])
			return errors.New(msg)
		} else {
			if ma != "" {
				t, err := time.Parse(time.RFC3339, ma)
				if err != nil {
					msg := fmt.Sprintf("failed to parse modified_at as time: %+v", t)
					return errors.New(msg)
				}
				t = t.UTC().Truncate(time.Second)
				m.ModifiedAt = &t
			}
		}
	}

	if modifiedBy, ok := rawStrings["modified_by"]; ok {
		if s, ok := modifiedBy.(string); !ok {
			msg := fmt.Sprintf("modified_by is not a string: %+v", rawStrings["modified_by"])
			return errors.New(msg)
		} else {
			m.ModifiedBy = s
		}
	}

	if name, ok := rawStrings["name"]; ok {
		if s, ok := name.(string); !ok {
			msg := fmt.Sprintf("name is not a string: %+v", rawStrings["name"])
			return errors.New(msg)
		} else {
			m.Name = s
		}
	}

	if scheduleExpression, ok := rawStrings["schedule_expression"]; ok {
		if s, ok := scheduleExpression.(string); !ok {
			msg := fmt.Sprintf("schedule_expression is not a string: %+v", rawStrings["schedule_expression"])
			return errors.New(msg)
		} else {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			_, err := parser.Parse(s)
			if err != nil {
				msg := fmt.Sprintf("schedule_expression is not a valid cron expression: '%s': %s", s, err)
				return errors.New(msg)
			}
			m.ScheduleExpression = s
		}
	}

	if enabled, ok := rawStrings["enabled"]; ok {
		if s, ok := enabled.(bool); !ok {
			msg := fmt.Sprintf("enabled is not a bool: %+v", rawStrings["enabled"])
			return errors.New(msg)
		} else {
			m.Enabled = s
		}
	}

	return nil
}

// MarshalJSON is a custom JSON marshaller for a job
func (m Job) MarshalJSON() ([]byte, error) {
	modifiedAt := ""
	if m.ModifiedAt != nil {
		modifiedAt = m.ModifiedAt.UTC().Truncate(time.Second).Format(time.RFC3339)
	}

	job := struct {
		Description        string            `json:"description"`
		Details            map[string]string `json:"details"`
		Group              string            `json:"group"`
		ID                 string            `json:"id"`
		ModifiedAt         string            `json:"modified_at"`
		ModifiedBy         string            `json:"modified_by"`
		Name               string            `json:"name"`
		ScheduleExpression string            `json:"schedule_expression"`
		Enabled            bool              `json:"enabled"`
	}{m.Description, m.Details, m.Group, m.ID, modifiedAt, m.ModifiedBy, m.Name, m.ScheduleExpression, m.Enabled}

	return json.Marshal(job)
}

package jobs

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestJobUnmarshalJSON(t *testing.T) {
	var rawJob = []byte(`
	{
		"id": "08d754ba-8540-4fdc-92f3-47950c1cdb1c",
		"description": "Alien sightings",
		"details": {
			"instance_id": "i-derpderpderp"
		},
		"group": "folder1",
		"modified_at": "2015-11-21T04:19:01.123Z",
		"modified_by": "zbrannigan",
		"name": "alien-sightings-dataset",
		"schedule_expression": "@hourly",
		"enabled": true
	}`)

	var modifiedAt, _ = time.Parse(time.RFC3339, "2015-11-21T04:19:01.000Z")
	var testJob = &Job{
		Description: "Alien sightings",
		Details: map[string]string{
			"instance_id": "i-derpderpderp",
		},
		Group:              "folder1",
		ID:                 "08d754ba-8540-4fdc-92f3-47950c1cdb1c",
		ModifiedAt:         &modifiedAt,
		ModifiedBy:         "zbrannigan",
		Name:               "alien-sightings-dataset",
		ScheduleExpression: "@hourly",
		Enabled:            true,
	}

	out := &Job{}
	err := out.UnmarshalJSON(rawJob)
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(out, testJob) {
		t.Errorf("expected: %+v,\n got %+v\n", testJob, out)
	}

	// bad json
	if err := out.UnmarshalJSON([]byte("{")); err == nil {
		t.Error("expected error for bad json, got nil")
	}

	// description type
	if err := out.UnmarshalJSON([]byte(`{"description":false}`)); err == nil {
		t.Error("expected error for bad description, got nil")
	}

	// details type
	if err := out.UnmarshalJSON([]byte(`{"details":false}`)); err == nil {
		t.Error("expected error for bad details type, got nil")
	}

	// details empty
	if err := out.UnmarshalJSON([]byte(`"details": {}`)); err == nil {
		t.Error("expected error for bad details, got nil")
	}

	// details value of wrong type
	if err := out.UnmarshalJSON([]byte(`{"details":{"foo": false}}`)); err == nil {
		t.Error("expected error for bad details, got nil")
	}

	// group type
	if err := out.UnmarshalJSON([]byte(`{"group":false}`)); err == nil {
		t.Error("expected error for bad group, got nil")
	}

	// id type
	if err := out.UnmarshalJSON([]byte(`{"id":false}`)); err == nil {
		t.Error("expected error for bad id, got nil")
	}

	// modified_at type
	if err := out.UnmarshalJSON([]byte(`{"modified_at":false}`)); err == nil {
		t.Error("expected error for bad modified_at, got nil")
	}

	// modified_at date type
	if err := out.UnmarshalJSON([]byte(`{"modified_at":"12345"}`)); err == nil {
		t.Error("expected error for bad modified_at, got nil")
	}

	// modified_by type
	if err := out.UnmarshalJSON([]byte(`{"modified_by":false}`)); err == nil {
		t.Error("expected error for bad modified_by, got nil")
	}

	// name type
	if err := out.UnmarshalJSON([]byte(`{"name":false}`)); err == nil {
		t.Error("expected error for bad name, got nil")
	}

	// schedule_expression invalid
	if err := out.UnmarshalJSON([]byte(`{"schedule_expression":""}`)); err == nil {
		t.Error("expected error for bad schedule_expression, got nil")
	}

	// schedule_expression type
	if err := out.UnmarshalJSON([]byte(`{"schedule_expression":false}`)); err == nil {
		t.Error("expected error for bad schedule_expression, got nil")
	}

	// enabled type
	if err := out.UnmarshalJSON([]byte(`{"enabled":"false"}`)); err == nil {
		t.Error("expected error for bad enabled, got nil")
	}
}

func TestMetadataMarshalJSON(t *testing.T) {
	type test struct {
		input  Job
		output []byte
		err    error
	}

	modifiedAt, _ := time.Parse(time.RFC3339, "2015-11-21T04:19:01.123Z")
	tests := []test{
		{
			Job{},
			[]byte(`{"account":"","description":"","details":null,"group":"","id":"","modified_at":"","modified_by":"","name":"","schedule_expression":"","enabled":false}`),
			nil,
		},
		{
			Job{
				Account:     "foocct",
				Description: "Alien sightings",
				Details: map[string]string{
					"instance_id": "i-derpderpderp",
				},
				Group:              "folder1",
				ID:                 "08d754ba-8540-4fdc-92f3-47950c1cdb1c",
				ModifiedAt:         &modifiedAt,
				ModifiedBy:         "kkroker",
				Name:               "alien-sightings-dataset",
				ScheduleExpression: "cron()",
				Enabled:            true,
			},
			[]byte(`{"account":"foocct","description":"Alien sightings","details":{"instance_id":"i-derpderpderp"},"group":"folder1","id":"08d754ba-8540-4fdc-92f3-47950c1cdb1c","modified_at":"2015-11-21T04:19:01Z","modified_by":"kkroker","name":"alien-sightings-dataset","schedule_expression":"cron()","enabled":true}`),
			nil,
		},
	}

	for _, tst := range tests {
		out, err := tst.input.MarshalJSON()
		if tst.err == nil && err != nil {
			t.Errorf("expected nil error, got %s", err)
		} else if tst.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", tst.err)
		}

		if !bytes.Equal(out, tst.output) {
			t.Errorf("expected: %s, got %s", string(tst.output), string(out))
		}
	}
}

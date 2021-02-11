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

func TestJob_NextRun(t *testing.T) {
	testTime, _ := time.Parse(time.RFC3339, "2015-11-21T04:19:01.123Z")
	hourlyTime, _ := time.Parse(time.RFC3339, "2015-11-21T05:00:00.000Z")
	allTheStars, _ := time.Parse(time.RFC3339, "2015-11-21T04:20:00.000Z")
	everyFive, _ := time.Parse(time.RFC3339, "2015-11-21T04:20:00.000Z")
	type fields struct {
		Account            string
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
	type args struct {
		t time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *time.Time
		wantErr bool
	}{
		{
			name:    "empty schedule",
			wantErr: true,
		},
		{
			name: "bad schedule words",
			fields: fields{
				ScheduleExpression: "@everybluemoon",
			},
			wantErr: true,
		},
		{
			name: "bad schedule stars",
			fields: fields{
				ScheduleExpression: "* *",
			},
			wantErr: true,
		},
		{
			name: "hourly",
			fields: fields{
				ScheduleExpression: "@hourly",
			},
			args: args{
				t: testTime,
			},
			want: &hourlyTime,
		},
		{
			name: "all the stars",
			fields: fields{
				ScheduleExpression: "* * * * *",
			},
			args: args{
				t: testTime,
			},
			want: &allTheStars,
		},
		{
			name: "every five minutes",
			fields: fields{
				ScheduleExpression: "*/5 * * * *",
			},
			args: args{
				t: testTime,
			},
			want: &everyFive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Job{
				Account:            tt.fields.Account,
				Description:        tt.fields.Description,
				Details:            tt.fields.Details,
				Enabled:            tt.fields.Enabled,
				ID:                 tt.fields.ID,
				ModifiedBy:         tt.fields.ModifiedBy,
				ModifiedAt:         tt.fields.ModifiedAt,
				Name:               tt.fields.Name,
				Group:              tt.fields.Group,
				ScheduleExpression: tt.fields.ScheduleExpression,
			}
			got, err := j.NextRun(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("Job.NextRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Job.NextRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

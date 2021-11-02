package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewTaskRunner(t *testing.T) {
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *TaskRunner
		wantErr bool
	}{
		{
			name:    "nil config",
			wantErr: true,
		},
		{
			name: "empty config",
			args: args{
				config: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "endpoint wrong type",
			args: args{
				config: map[string]interface{}{
					"endpoint": 123,
				},
			},
			wantErr: true,
		},
		{
			name: "endpoint_template wrong type",
			args: args{
				config: map[string]interface{}{
					"endpoint_template": 123,
				},
			},
			wantErr: true,
		},
		{
			name: "encrypt is wrong type",
			args: args{
				config: map[string]interface{}{
					"endpoint": "http://127.0.0.1:8080/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks",
					"encrypt":  12345,
				},
			},
			want: &TaskRunner{
				AuthHeader: "X-Auth-Token",
				Encrypt:    true,
				Endpoint:   "http://127.0.0.1:8080/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks",
			},
		},
		{
			name: "endpoint and endpoint_template",
			args: args{
				config: map[string]interface{}{
					"endpoint":         "http://127.0.0.1:8080/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks",
					"endpointTemplate": "http://127.0.0.1:8080/v1/ecs/{{.Account}}/cluster/{{.Cluster}}/taskdefs/{{ .Name }}/tasks",
				},
			},
			want: &TaskRunner{
				AuthHeader:       "X-Auth-Token",
				Encrypt:          true,
				Endpoint:         "http://127.0.0.1:8080/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks",
				EndpointTemplate: "http://127.0.0.1:8080/v1/ecs/{{.Account}}/cluster/{{.Cluster}}/taskdefs/{{ .Name }}/tasks",
			},
		},
		{
			name: "endpoint string",
			args: args{
				config: map[string]interface{}{
					"endpoint": "http://127.0.0.1:8080/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks",
				},
			},
			want: &TaskRunner{
				AuthHeader: "X-Auth-Token",
				Encrypt:    true,
				Endpoint:   "http://127.0.0.1:8080/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks",
			},
		},
		{
			name: "endpointTemplate string",
			args: args{
				config: map[string]interface{}{
					"endpointTemplate": "http://127.0.0.1:8080/v1/ecs/{{.Account}}/cluster/{{.Cluster}}/taskdefs/{{ .Name }}/tasks",
				},
			},
			want: &TaskRunner{
				AuthHeader:       "X-Auth-Token",
				Encrypt:          true,
				EndpointTemplate: "http://127.0.0.1:8080/v1/ecs/{{.Account}}/cluster/{{.Cluster}}/taskdefs/{{ .Name }}/tasks",
			},
		},
		{
			name: "full config",
			args: args{
				config: map[string]interface{}{
					"endpointTemplate": "http://127.0.0.1:8080/v1/ecs/{{.Account}}/cluster/{{.Cluster}}/taskdefs/{{ .Name }}/tasks",
					"token":            "123456789890",
					"encrypt_token":    true,
					"auth_header":      "X-Top-Sekret",
				},
			},
			want: &TaskRunner{
				AuthHeader:       "X-Top-Sekret",
				Encrypt:          true,
				EndpointTemplate: "http://127.0.0.1:8080/v1/ecs/{{.Account}}/cluster/{{.Cluster}}/taskdefs/{{ .Name }}/tasks",
				Token:            "123456789890",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTaskRunner(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskRunner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTaskRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskRunner_Run(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, "unexpected method "+r.Method+"expected "+http.MethodPost)
			return
		}

		tok := r.Header.Get("X-Auth-Token")
		t.Logf("got token header with request: %s", tok)

		if err := bcrypt.CompareHashAndPassword([]byte(tok), []byte("my-awesome-token")); err != nil {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "bad token")
			return
		}

		if r.URL.Path != "/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks" {
			msg := fmt.Sprintf("bad path %s", r.URL.Path)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		inputPayload := struct {
			Count     int
			StartedBy string
		}{}
		err := json.NewDecoder(r.Body).Decode(&inputPayload)
		if err != nil {
			http.Error(w, "cannot decode body into input", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	type fields struct {
		Endpoint         string
		EndpointTemplate string
		Token            string
		Encrypt          bool
		AuthHeader       string
	}
	type args struct {
		ctx        context.Context
		account    string
		parameters interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "empty input",
			wantErr: true,
		},
		{
			name: "wrong params type",
			args: args{
				ctx:        context.TODO(),
				account:    "acct",
				parameters: "foobar",
			},
			wantErr: true,
		},
		{
			name: "missing task_action parameter",
			args: args{
				ctx:        context.TODO(),
				account:    "acct",
				parameters: map[string]string{"foo": "bar"},
			},
			wantErr: true,
		},
		{
			name: "unknown task_action parameter",
			args: args{
				ctx:        context.TODO(),
				account:    "acct",
				parameters: map[string]string{"task_action": "bar"},
			},
			wantErr: true,
		},
		{
			name: "valid task_action, missing task_cluster",
			args: args{
				ctx:     context.TODO(),
				account: "acct",
				parameters: map[string]string{
					"task_action": "run",
					"task_name":   "foo",
					"count":       "123",
				},
			},
			wantErr: true,
		},
		{
			name: "valid task_action, task_cluster, missing task_name",
			args: args{
				ctx:     context.TODO(),
				account: "acct",
				parameters: map[string]string{
					"task_action":  "run",
					"task_cluster": "foo",
					"count":        "123",
				},
			},
			wantErr: true,
		},
		{
			name: "valid task_action, task_cluster, task_name, missing count",
			args: args{
				ctx:     context.TODO(),
				account: "acct",
				parameters: map[string]string{
					"task_action":  "run",
					"task_cluster": "foo",
					"task_name":    "foo",
				},
			},
			wantErr: true,
		},
		{
			name: "valid task_action, task_cluster, task_name, invalid count",
			args: args{
				ctx:     context.TODO(),
				account: "acct",
				parameters: map[string]string{
					"task_action":  "run",
					"task_cluster": "foo",
					"task_name":    "foo",
					"count":        "three",
				},
			},
			wantErr: true,
		},
		{
			name: "valid params, count of 0",
			fields: fields{
				AuthHeader: "X-Auth-Token",
				Encrypt:    true,
				Endpoint:   fmt.Sprintf("%s/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks", ts.URL),
				Token:      "my-awesome-token",
			},
			args: args{
				ctx:     context.TODO(),
				account: "acct",
				parameters: map[string]string{
					"task_action":  "run",
					"task_cluster": "foo",
					"task_name":    "foo",
					"count":        "0",
				},
			},
			wantErr: true,
		},
		{
			name: "valid params",
			fields: fields{
				AuthHeader: "X-Auth-Token",
				Encrypt:    true,
				Endpoint:   fmt.Sprintf("%s/v1/ecs/acct1/cluster/clu1/taskdefs/td1/tasks", ts.URL),
				Token:      "my-awesome-token",
			},
			args: args{
				ctx:     context.TODO(),
				account: "acct",
				parameters: map[string]string{
					"task_action":  "run",
					"task_cluster": "foo",
					"task_name":    "foo",
					"count":        "1",
				},
			},
			want: "successfully submitted run task foo/foo with count 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TaskRunner{
				Endpoint:         tt.fields.Endpoint,
				EndpointTemplate: tt.fields.EndpointTemplate,
				Token:            tt.fields.Token,
				Encrypt:          tt.fields.Encrypt,
				AuthHeader:       tt.fields.AuthHeader,
			}
			got, err := r.Run(tt.args.ctx, tt.args.account, tt.args.parameters)
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskRunner.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TaskRunner.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

package jobs

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type DummyRunner struct {
	Template string
}

// NewDummyRunner creates a new dummy runner that doesn't do anything, but requires a
// template and returns that executed template when called
func NewDummyRunner(config map[string]interface{}) (*DummyRunner, error) {
	log.Debug("creating new dummy job runner")

	var template string
	if v, ok := config["template"].(string); ok {
		template = v
	}

	if template == "" {
		return nil, errors.New("template cannot be empty")
	}

	return &DummyRunner{Template: template}, nil
}

// Run executes the DummyRunner, ignoring the parameters
func (r *DummyRunner) Run(ctx context.Context, account string, parameters interface{}) (string, error) {
	if account == "" {
		return "", errors.New("account is required")
	}

	log.Infof("running dummy runner %+v in account %s,  with parameters %+v", r, account, parameters)

	input := struct {
		Account string
	}{account}

	tmpl, err := template.New("dummy").Parse(r.Template)
	if err != nil {
		return "", NewRunnerError(ErrPreExecFailure, "template parsing failed", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, &input); err != nil {
		return "", NewRunnerError(ErrPreExecFailure, "template parsing failed", err)
	}

	msg := out.String()
	log.Debugf("output of template: %s", msg)

	return msg, nil
}

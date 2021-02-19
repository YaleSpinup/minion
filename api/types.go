package api

import (
	"github.com/YaleSpinup/minion/cloudwatchlogs"
	"github.com/YaleSpinup/minion/jobs"
)

type JobsResponse struct {
	Job  *jobs.Job                `json:"job"`
	Tags []*tag                   `json:"tags"`
	Log  *cloudwatchlogs.LogGroup `json:"log"`
	Next string                   `json:"next"`
}

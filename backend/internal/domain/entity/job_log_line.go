package entity

import "time"

const (
	LogStreamStdout = "stdout"
	LogStreamStderr = "stderr"
)

type JobLogLine struct {
	ID             int64
	OrganizationID string
	RepositoryID   string
	RunID          string
	JobID          string
	StepIndex      int
	LineNumber     int64
	Stream         string
	Text           string
	CreatedAt      time.Time
}

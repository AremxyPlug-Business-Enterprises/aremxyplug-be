package timehelper

import (
	"time"
)

//go:generate mockgen -source=timehelper.go -destination=../../mocks/timehelper_mock.go -package=mocks
type TimeHelper interface {
	Now() time.Time
}

const (
	// date templates
	DobTemplate       = "02-01-2006"
	MambuDateTemplate = "2006-01-02"
)

// Ensure implementation of TimeHelper interface
var _ TimeHelper = (*timeHelper)(nil)

type timeHelper struct{}

// Now returns the current local time.
func (t timeHelper) Now() time.Time {
	return time.Now()
}

// New return a new instance of a core definition for TimeHelper interface
func New() TimeHelper {
	return &timeHelper{}
}

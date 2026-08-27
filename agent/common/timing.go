package common

import (
	"strconv"
	"time"

	"github.com/jollaman999/utils/logger"
)

// LogElapsed reports how long one collection step took. Collection latency is
// uneven across sub-collectors -- some shell out per process, some fetch over
// the network -- so each step is timed individually to show where an import
// actually spends its time.
//
// detail carries step-specific context (an item count, a payload size) and may
// be empty.
func LogElapsed(scope string, step string, start time.Time, detail string) {
	msg := "TIMING: " + scope + ": " + step + " took " +
		time.Since(start).Round(time.Millisecond).String()
	if detail != "" {
		msg += " (" + detail + ")"
	}

	logger.Println(logger.INFO, false, msg)
}

// CountDetail formats an item count for LogElapsed's detail argument.
func CountDetail(count int) string {
	return strconv.Itoa(count) + " items"
}

package reporting

import (
	"testing"
)

func TestEmit(t *testing.T) {
	reporter := NewReporter(false, nil, nil)
	reporter.TaskStart("blah", "baz")
	reporter.TaskDone()
}

// github.com/seamia/memory

package memory

import (
	"fmt"

	"github.com/nontechno/link"
)

const (
	WarningFunc = "seamia/memory:warning"
	ReportFunc  = "seamia/memory:report"
	TraceFunc   = "seamia/memory:trace"
)

var (
	warning func(string, ...interface{})
	report  func(string, ...interface{})
	trace   func(string, ...interface{})
)

func noop(string, ...interface{}) {}

func noop2(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func init() {
	link.Link(&warning, WarningFunc, noop)
	link.Link(&report, ReportFunc, noop)
	link.Link(&trace, TraceFunc, noop)
}

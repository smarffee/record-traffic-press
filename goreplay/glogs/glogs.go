package glogs

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var PreviousDebugTime = time.Now()
var DebugMutex sync.Mutex

// Debug take an effect only if --verbose greater than 0 is specified
func Debug(level int, args ...interface{}) {
	if 1 >= level {
		DebugMutex.Lock()
		defer DebugMutex.Unlock()
		now := time.Now()
		diff := now.Sub(PreviousDebugTime)
		PreviousDebugTime = now
		fmt.Fprintf(os.Stderr, "[DEBUG][elapsed %s]: ", diff)
		fmt.Fprintln(os.Stderr, args...)
	}
}

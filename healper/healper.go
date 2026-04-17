package healper

import (
	"fmt"
	"sync/atomic"
	"time"
)


var Counter uint64 // Counter is a global variable that is used to generate unique IDs for containers. It is incremented atomically to ensure thread safety.

func GenrateRandomUUid() string { // GenrateRandomUUid generates a unique ID by combining an atomic counter with the current time in nanoseconds, and returns it as a hexadecimal string
	id := atomic.AddUint64(&Counter, 1)
	id = id + uint64(time.Now().UnixNano())
	return fmt.Sprintf("%x",id)
}



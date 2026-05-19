package process

import (
	"fmt"
	"os"
	"strconv"
)

func setupCgroup(pid int) error {
	base := "/sys/fs/cgroup/ctr-" + strconv.Itoa(pid)

	if err := os.Mkdir(base, 0o755); err != nil {
		return fmt.Errorf("mkdir cgroup: %w", err)
	}

	limits := []struct{ file, value string }{
		{"memory.max", "67108864"},  // 64 MB RAM
		{"cpu.max", "20000 100000"}, // 20 % CPU (20 ms per 100 ms period)
		{"pids.max", "64"},          // max 64 processes
	}
	for _, l := range limits {
		if err := os.WriteFile(base+"/"+l.file, []byte(l.value), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", l.file, err)
		}
	}

	return os.WriteFile(base+"/cgroup.procs", []byte(strconv.Itoa(pid)), 0o644)
}

// Package healper provide an function to genrate unique uuids for users and containers
package healper

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Status int


const FOlderpath = "~/.config/sherpa"

const (
	Active Status = iota
	Stopped
	Pending
)

var Counter uint64 // Counter is a global variable that is used to generate unique IDs for containers. It is incremented atomically to ensure thread safety.

func GenrateRandomUUid() string { // GenrateRandomUUid generates a unique ID by combining an atomic counter with the current time in nanoseconds, and returns it as a hexadecimal string
	id := atomic.AddUint64(&Counter, 1)
	id = id + uint64(time.Now().UnixNano())
	return fmt.Sprintf("%x", id)
}

func GenrateNetworkid() string {
	id := uint8(time.Now().UnixNano())
	return fmt.Sprintf("%x", id)
}

func PrintBanner() {
	fmt.Println(`
   ███████╗██╗   ██╗██████╗ ███████╗██████╗ ███████╗ █████╗ ███╗   ██╗██████╗ 
   ██╔════╝██║   ██║██╔══██╗██╔════╝██╔══██╗██╔════╝██╔══██╗████╗  ██║██╔══██╗
   ███████╗██║   ██║██████╔╝█████╗  ██████╔╝███████╗███████║██╔██╗ ██║██║  ██║
   ╚════██║██║   ██║██╔═══╝ ██╔══╝  ██╔══██╗╚════██║██╔══██║██║╚██╗██║██║  ██║
   ███████║╚██████╔╝██║     ███████╗██║  ██║███████║██║  ██║██║ ╚████║██████╔╝
   ╚══════╝ ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝ 

  ⚡  Supersand — Secure Sandbox Runtime  ⚡
`)
}

func GetTotalRAM() int {
	data, _ := os.ReadFile("/proc/meminfo")
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal") {
			fields := strings.Fields(line)
			kb, _ := strconv.Atoi(fields[1])
			return kb / 1024
		}
	}
	return 0
}

func DecideLimits() int {
	totalRAM := GetTotalRAM()
	cpuCores := runtime.NumCPU()

	// Reserve 30% for system
	usableRAM := int(float64(totalRAM) * 0.7)

	// Decide per container usage
	memPerContainer := 256 // MB (tunable)

	maxByRAM := usableRAM / memPerContainer

	// CPU logic (0.5 core per container)
	cpuPerContainer := 0.5
	maxByCPU := int(float64(cpuCores) / cpuPerContainer)

	// Final limit = min of both
	maxContainers := min(maxByRAM, maxByCPU)
	return maxContainers
}




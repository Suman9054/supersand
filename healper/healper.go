// Package healper provide an function to genrate unique uuids for users and containers
package healper

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Status int

const FOlderpath = "~/.config/supersand"

const (
	Active Status = iota
	Stopped
	Pending
)

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

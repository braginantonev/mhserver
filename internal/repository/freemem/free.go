package freemem

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Get free memory in bytes from /proc/meminfo
// If an error is received return 0
func GetAvailableMemory() uint64 {
	file, err := os.OpenInRoot("/proc/", "meminfo")
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Expected str: `MemAvailable: xxx kB`

		txt := scanner.Text()
		if txt[:12] != "MemAvailable" {
			continue
		}

		// txt[8:len(txt)-3] - slice without MemAvailable: and Kb with space
		num, err := strconv.Atoi(strings.Trim(txt[13:len(txt)-3], " "))
		if err != nil {
			return 0
		}

		return uint64(num) * 1024
	}

	return 0
}

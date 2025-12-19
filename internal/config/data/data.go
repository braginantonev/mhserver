package dataconfig

import (
	"fmt"
	"log/slog"
	"strconv"
)

var (
	ram_sizes = map[byte]uint64{
		'B': 1,
		'K': 1024,
		'M': 1024 * 1024,
		'G': 1024 * 1024 * 1024,
		'T': 1024 * 1024 * 1024 * 1024,
	}
)

const (
	STANDARD_RAM_SIZE  byte    = 'M'
	STANDARD_RAM_VALUE float64 = 500.1
)

type DataServiceConfig struct {
	WorkspacePath string // User files path
	AvailableRAM  uint64

	//Todo: Upload semaphore
}

// available ram in form: "1.2M", "0.1K" etc
func NewDataServerConfig(workspace_path string, available_ram string) DataServiceConfig {
	cfg := DataServiceConfig{
		WorkspacePath: workspace_path,
	}

	if available_ram == "" {
		available_ram = fmt.Sprint(STANDARD_RAM_VALUE, string(STANDARD_RAM_SIZE))
	}

	ram_size, ok := ram_sizes[available_ram[len(available_ram)-1]]
	if !ok {
		slog.Warn("Server available ram size not set!", slog.String("standard value", string(STANDARD_RAM_SIZE)))
		ram_size = ram_sizes[STANDARD_RAM_SIZE]
	}

	ram_value, err := strconv.ParseFloat(available_ram[:len(available_ram)-1], 64)
	if err != nil {
		slog.Warn("Bad ram value.", slog.Float64("standard value", STANDARD_RAM_VALUE))
		ram_value = STANDARD_RAM_VALUE
	}

	cfg.AvailableRAM = uint64(ram_value) * ram_size
	return cfg
}

type DataHandlerConfig struct {
	ServiceConfig    DataServiceConfig
	MaxRequestsCount int
}

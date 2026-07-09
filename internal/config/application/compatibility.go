package appconfig

import "fmt"

func (cfg *ApplicationConfig) checkMemoryIncompatibility() error {
	min_allocated_memory := cfg.Memory.MaxChunkSize + 1024 // 1024 - http request meta
	for n, s := range cfg.SubServers {
		// check low allocated memory
		if s.Extra.AllocatedMemory < min_allocated_memory {
			return fmt.Errorf("allocated memory for %s subserver is very low (min. required - %d bytes)", n, min_allocated_memory)
		}

		// check big allocated memory
		if s.Extra.AllocatedMemory > cfg.Memory.Allocated {
			return fmt.Errorf("allocated memory for %s subserver is bigger than server have", n)
		}
	}

	return nil
}

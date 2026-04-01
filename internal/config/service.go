package config

import "time"

type ServiceName string

type RequestsConfig struct {
	LimiterInterval time.Duration
	MaxInInterval   int
}

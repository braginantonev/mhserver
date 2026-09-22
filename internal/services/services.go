package services

import "github.com/braginantonev/mhserver/internal/config"

type ServiceName string

type Dependencies map[ServiceName]config.ServerSocket

type ServiceConfig struct {
	ServiceName  ServiceName `toml:"-"`
	Dependencies Dependencies
}

type Service interface {
	InitDependencies() error
}

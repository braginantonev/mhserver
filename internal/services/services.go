package services

type ServiceName string

type ServiceConfig interface {
	initDependencies() error

	Init() error
	GetServiceName() ServiceName
}

type Service interface {
	GetConfig() ServiceConfig
}

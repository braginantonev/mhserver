package application

import "errors"

var (
	ErrConfigurationNotFound error = errors.New("configuration file not found! Please run `setup.sh` and setup your server")
	ErrEnvironmentNotFound   error = errors.New("env not found! Please run `setup.sh`")
)

package application

import "errors"

var (
	ERR_CONF_NF          error = errors.New("Configuration file not found! Please run `setup.sh` and setup your server")
	ERR_ENV_NF           error = errors.New("env not found! Please run `setup.sh`")
	ERR_BAD_START_SERVER error = errors.New("Failed to start server")
)

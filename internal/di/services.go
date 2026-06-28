package di

import (
	"database/sql"

	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/service/auth"
)

func SetupAuthService(app_cfg appconfig.ApplicationConfig, db *sql.DB) *auth.AuthService {
	available_services := make([]string, 0, len(app_cfg.SubServers))
	for sub := range app_cfg.SubServers {
		available_services = append(available_services, sub)
	}

	return auth.NewAuthService(auth.AuthConfig{
		JWTSignature:  app_cfg.JWTSignature,
		WorkspacePath: app_cfg.WorkspacePath,
		UserCatalogs:  available_services[1:],
	}, db)
}

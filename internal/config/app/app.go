package appconfig

type ApplicationConfig struct {
	ServerName    string `toml:"server_name"`
	WorkspacePath string `toml:"workspace_path"`
	JWTSignature  string `toml:"jwt_signature"`
	DB_Pass       string `toml:"db_pass"`
	SubServers    map[string]SubServer
}

type SubServer struct {
	Enabled  bool
	HostName string
	IP       string
	Port     string
}

package auth

type AuthConfig struct {
	JWTSignature  string
	WorkspacePath string
	UserCatalogs  []string
}

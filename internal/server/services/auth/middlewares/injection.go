package middlewares

type Config struct {
	JWTSignature string
}

type AuthMiddleware struct {
	cfg Config
}

func NewAuthMiddleware(cfg Config) AuthMiddleware {
	return AuthMiddleware{
		cfg: cfg,
	}
}

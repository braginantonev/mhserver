package contextkeys

type ContextKey string

const (
	USERNAME        ContextKey = "username"
	InternalRequest ContextKey = "internal-request"
)

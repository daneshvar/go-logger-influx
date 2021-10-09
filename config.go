package loggerinflux

type Config struct {
	Caller *bool
	Stack  *string
	Level  *string
	Scope  map[string]string

	URL     string
	Token   string
	Org     string
	Bucket  string
	AppName string
}

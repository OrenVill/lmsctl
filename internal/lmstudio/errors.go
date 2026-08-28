package lmstudio

import "fmt"

// ErrUnreachable indicates the LM Studio server could not be reached at all
// (connection refused, DNS failure, timeout).
type ErrUnreachable struct {
	Host string
	Err  error
}

func (e *ErrUnreachable) Error() string {
	return fmt.Sprintf("could not reach LM Studio at %s: %v\n"+
		"check that LM Studio is running on that machine and that "+
		"\"Serve on Local Network\" is enabled in its Developer settings", e.Host, e.Err)
}

func (e *ErrUnreachable) Unwrap() error { return e.Err }

// ErrUnauthorized indicates the server rejected the request due to a
// missing or incorrect API token.
type ErrUnauthorized struct{}

func (e *ErrUnauthorized) Error() string {
	return "LM Studio rejected the request: missing or incorrect API token\n" +
		"set the correct token with --token, the LMSCTL_TOKEN environment variable, " +
		"or the \"token:\" field in ~/.config/lmsctl/config.yaml"
}

// ErrModelNotFound indicates the requested model key does not match any
// downloaded model.
type ErrModelNotFound struct {
	Model string
}

func (e *ErrModelNotFound) Error() string {
	return fmt.Sprintf("no downloaded model matches %q — run 'lmsctl models' to see available models", e.Model)
}

// ErrModelNotLoaded indicates an unload was requested for a model that has
// no loaded instance.
type ErrModelNotLoaded struct {
	Model string
}

func (e *ErrModelNotLoaded) Error() string {
	return fmt.Sprintf("%q is not currently loaded — run 'lmsctl status' to see what is", e.Model)
}

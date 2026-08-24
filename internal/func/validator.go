package funcs

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"fnexec/internal/model"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

// Validate checks that a function definition can be registered safely.
func Validate(fn *model.Function) error {
	if fn == nil {
		return errors.New("function must not be nil")
	}
	if !namePattern.MatchString(fn.Name) {
		return errors.New("function name must match ^[a-z][a-z0-9_.-]{0,63}$")
	}
	if fn.Handler == nil {
		return errors.New("function handler must not be nil")
	}
	if fn.Timeout <= 0 || fn.Timeout > time.Minute {
		return errors.New("function timeout must be within (0, 1m]")
	}
	if fn.MaxRetries < 0 || fn.MaxRetries > 5 {
		return errors.New("function max retries must be within [0, 5]")
	}
	if fn.MinInstances < 0 || fn.MaxInstances < fn.MinInstances || fn.MaxInstances > 16 {
		return errors.New("function instance bounds are invalid")
	}
	return nil
}

// NormalizeName trims and lower-cases a function name before lookup.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

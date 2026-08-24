package funcs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fnexec/internal/model"
)

type echoRequest struct {
	Text string `json:"text"`
}

type concatRequest struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// RegisterBuiltins installs the demo functions used by the console and probes.
func RegisterBuiltins(r *Registry) error {
	echo := &model.Function{
		Name:    "echo",
		Timeout: 2 * time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			var req echoRequest
			if err := json.Unmarshal(payload, &req); err != nil {
				return nil, fmt.Errorf("decode echo payload: %w", err)
			}
			out, _ := json.Marshal(map[string]string{"text": req.Text})
			return out, nil
		},
	}
	concat := &model.Function{
		Name:    "concat",
		Timeout: 2 * time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			var req concatRequest
			if err := json.Unmarshal(payload, &req); err != nil {
				return nil, fmt.Errorf("decode concat payload: %w", err)
			}
			out, _ := json.Marshal(map[string]string{"joined": req.Left + req.Right})
			return out, nil
		},
	}
	upper := &model.Function{
		Name:    "upper",
		Timeout: 2 * time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			text := strings.ToUpper(strings.TrimSpace(string(payload)))
			return []byte(text), nil
		},
	}
	failif := &model.Function{
		Name:    "failif",
		Timeout: 2 * time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			if strings.Contains(string(payload), "boom") {
				return nil, fmt.Errorf("boom requested")
			}
			return payload, nil
		},
	}
	checksum := &model.Function{
		Name:    "checksum",
		Timeout: 2 * time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			out, _ := json.Marshal(map[string]string{"checksum": model.HashID(string(payload))})
			return out, nil
		},
	}
	for _, fn := range []*model.Function{echo, concat, upper, failif, checksum} {
		if err := r.Register(fn); err != nil {
			return err
		}
	}
	return nil
}

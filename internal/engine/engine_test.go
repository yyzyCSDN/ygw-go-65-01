package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"fnexec/internal/model"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	eng, err := New(Options{
		HTTPAddr:       "127.0.0.1:0",
		BatchSize:      4,
		Workers:        2,
		DefaultTimeout: 500 * time.Millisecond,
		HandleLimit:    16,
		ScaleInterval:  time.Hour,
		ConsoleHTML:    []byte("<html><body>console</body></html>"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

func TestEngineHealthz(t *testing.T) {
	eng := testEngine(t)
	if err := eng.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop(context.Background())
	resp, err := http.Get("http://" + eng.Addr() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz must return 200, got %d", resp.StatusCode)
	}
}

func TestEngineInvokeRoundTrip(t *testing.T) {
	eng := testEngine(t)
	if err := eng.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop(context.Background())
	body := bytes.NewBufferString(`{"func":"echo","payload":{"text":"hi"}}`)
	resp, err := http.Post("http://"+eng.Addr()+"/api/invoke", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("invoke must succeed, got %d: %s", resp.StatusCode, data)
	}
	var out struct {
		CallID string `json:"call_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.CallID == "" || out.Status != "queued" {
		t.Fatalf("unexpected invoke response: %+v", out)
	}
}

func TestEngineStatsAndConsole(t *testing.T) {
	eng := testEngine(t)
	if err := eng.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop(context.Background())
	resp, err := http.Get("http://" + eng.Addr() + "/api/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats must return 200, got %d", resp.StatusCode)
	}
	console, err := http.Get("http://" + eng.Addr() + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer console.Body.Close()
	if console.StatusCode != http.StatusOK {
		t.Fatalf("console must return 200, got %d", console.StatusCode)
	}
}

func TestEngineInvokeMethod(t *testing.T) {
	eng := testEngine(t)
	ev := model.NewEvent("e1", "upper", []byte("abc"))
	call, err := eng.Invoke(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if call.FuncName != "upper" || call.ID == "" {
		t.Fatalf("unexpected call: %+v", call)
	}
}

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"fnexec/internal/model"
)

type fakeInvoker struct{}

func (f *fakeInvoker) Invoke(ctx context.Context, ev model.Event) (*model.Call, error) {
	return model.NewCall("call-1", ev.FuncName, ev.Payload, ev.Time.Add(0)), nil
}

type fakeProvider struct{}

func (f *fakeProvider) Snapshot() Stats { return Stats{Version: "test"} }

type fakeResults struct{}

func (f *fakeResults) GetResult(callID string) *model.Result {
	return &model.Result{CallID: callID, Outcome: model.OutcomeSuccess}
}

func TestServerRoutes(t *testing.T) {
	srv := NewServer(&fakeInvoker{}, &fakeProvider{}, &fakeResults{}, []byte("<html>ok</html>"))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, path := range []string{"/healthz", "/api/stats", "/"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s must return 200, got %d", path, resp.StatusCode)
		}
	}
}

func TestServerInvokeAndCallLookup(t *testing.T) {
	srv := NewServer(&fakeInvoker{}, &fakeProvider{}, &fakeResults{}, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/invoke", http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("invoke with an empty body must be rejected")
	}

	callResp, err := http.Get(ts.URL + "/api/calls/call-1")
	if err != nil {
		t.Fatal(err)
	}
	callResp.Body.Close()
	if callResp.StatusCode != http.StatusOK {
		t.Fatalf("call lookup must return 200, got %d", callResp.StatusCode)
	}
}

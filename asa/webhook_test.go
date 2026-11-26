package xtremeasa

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	xtremeres "github.com/globalxtreme/go-core/v2/response"
)

const testToken = "asa_test_token"

func serve(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	t.Setenv("ASA_WEBHOOK_HOST", server.URL+"/")
	t.Setenv("ASA_WEBHOOK_TOKEN", testToken)
}

func TestPushSendsRequestAndReadsExecution(t *testing.T) {
	var got *http.Request
	var gotBody []byte

	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		gotBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":{"code":200,"message":"Success"},"result":{"executionUuid":"exec-1"}}`))
	})

	result := Webhook{
		Path:    "wo-created",
		Data:    map[string]interface{}{"woNumber": "WO-1"},
		Query:   url.Values{"source": {"crm"}},
		Headers: map[string]string{"X-Request-Id": "req-1"},
	}.Push()

	if result.StatusCode != http.StatusOK || result.ExecutionUUID != "exec-1" {
		t.Fatalf("result = %d %q", result.StatusCode, result.ExecutionUUID)
	}
	if got.Method != http.MethodPost || got.URL.Path != "/api/webhook/v1/asa-automations/wo-created" {
		t.Fatalf("request = %s %s", got.Method, got.URL.Path)
	}
	if got.URL.Query().Get("source") != "crm" {
		t.Fatalf("query = %q", got.URL.RawQuery)
	}
	if got.Header.Get("Authorization") != "Bearer "+testToken || got.Header.Get("X-Request-Id") != "req-1" {
		t.Fatalf("headers = %v", got.Header)
	}
	if got.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", got.Header.Get("Content-Type"))
	}

	var body map[string]interface{}
	if err := json.Unmarshal(gotBody, &body); err != nil || body["woNumber"] != "WO-1" {
		t.Fatalf("body = %s", gotBody)
	}
}

func TestTryPushOnlyNeedsPath(t *testing.T) {
	var got *http.Request

	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
	})

	if _, err := (Webhook{Path: "ping", Method: "get"}).TryPush(); err != nil {
		t.Fatalf("TryPush: %v", err)
	}

	if got.Method != http.MethodGet || got.URL.Path != "/api/webhook/v1/asa-automations/ping" {
		t.Fatalf("request = %s %s", got.Method, got.URL.Path)
	}
	if got.Header.Get("Content-Type") != "" {
		t.Fatal("content type must not be sent without data")
	}
}

func TestTryPushTokenCannotBeOverriddenByHeaders(t *testing.T) {
	var got *http.Request

	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
	})

	hook := Webhook{Path: "ping", Headers: map[string]string{"Authorization": "Bearer other"}}
	if _, err := hook.TryPush(); err != nil {
		t.Fatalf("TryPush: %v", err)
	}

	if got.Header.Get("Authorization") != "Bearer "+testToken {
		t.Fatalf("authorization = %q", got.Header.Get("Authorization"))
	}
}

func TestTryPushReturnsErrorOnFailureStatus(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":{"code":401,"message":"Webhook authentication failed","internalMsg":"unknown service token"}}`))
	})

	result, err := (Webhook{Path: "wo-created"}).TryPush()
	if err == nil {
		t.Fatal("expected error")
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", result.StatusCode)
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Webhook authentication failed") || !strings.Contains(err.Error(), "unknown service token") {
		t.Fatalf("error = %v", err)
	}
}

func TestTryPushKeepsRawWorkflowResponse(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"approved":true}`))
	})

	result, err := (Webhook{Path: "approve"}).TryPush()
	if err != nil {
		t.Fatalf("TryPush: %v", err)
	}

	var body struct {
		Approved bool `json:"approved"`
	}
	if err = result.JSON(&body); err != nil || !body.Approved || result.ExecutionUUID != "" {
		t.Fatalf("body = %s, execution = %q", result.Body, result.ExecutionUUID)
	}
}

func TestTryPushRequiresEnvironmentAndPath(t *testing.T) {
	t.Setenv("ASA_WEBHOOK_HOST", "")
	t.Setenv("ASA_WEBHOOK_TOKEN", testToken)

	if _, err := (Webhook{Path: "wo-created"}).TryPush(); err == nil || !strings.Contains(err.Error(), "ASA_WEBHOOK_HOST") {
		t.Fatalf("expected ASA_WEBHOOK_HOST error, got %v", err)
	}

	t.Setenv("ASA_WEBHOOK_HOST", "http://localhost")
	t.Setenv("ASA_WEBHOOK_TOKEN", "")

	if _, err := (Webhook{Path: "wo-created"}).TryPush(); err == nil || !strings.Contains(err.Error(), "ASA_WEBHOOK_TOKEN") {
		t.Fatalf("expected ASA_WEBHOOK_TOKEN error, got %v", err)
	}

	t.Setenv("ASA_WEBHOOK_TOKEN", testToken)

	if _, err := (Webhook{Path: "/"}).TryPush(); err == nil {
		t.Fatal("expected error without path")
	}
}

func TestTryPushTimesOut(t *testing.T) {
	release := make(chan struct{})
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })

	t.Setenv("ASA_WEBHOOK_TIMEOUT", "1")

	if _, err := (Webhook{Path: "slow"}).TryPush(); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPushPanicsWithResponseError(t *testing.T) {
	t.Setenv("ASA_WEBHOOK_HOST", "http://localhost")
	t.Setenv("ASA_WEBHOOK_TOKEN", testToken)

	defer func() {
		recovered := recover()

		resErr, ok := recovered.(*xtremeres.ResponseError)
		if !ok || resErr.Status.Code != http.StatusInternalServerError {
			t.Fatalf("recovered = %#v", recovered)
		}
	}()

	Webhook{}.Push()
}

func TestTryPushNormalizesPathSlashes(t *testing.T) {
	var got *http.Request

	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(`{}`))
	})

	for _, path := range []string{"wo-created", "/wo-created", "wo-created/", "//wo-created//", " /wo-created "} {
		if _, err := (Webhook{Path: path}).TryPush(); err != nil {
			t.Fatalf("TryPush(%q) error = %v", path, err)
		}

		if got.URL.Path != "/api/webhook/v1/asa-automations/wo-created" {
			t.Fatalf("TryPush(%q) path = %s", path, got.URL.Path)
		}
	}

	if _, err := (Webhook{Path: "/orders/created"}).TryPush(); err != nil {
		t.Fatalf("TryPush error = %v", err)
	}

	if got.URL.Path != "/api/webhook/v1/asa-automations/orders/created" {
		t.Fatalf("nested path = %s", got.URL.Path)
	}
}

func TestTryPushSendsExecutionMetadata(t *testing.T) {
	var got *http.Request

	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
	})

	_, err := Webhook{
		Path:          "wo-created",
		Headers:       map[string]string{"X-ASA-Created-By": "other"},
		Description:   "WO-1 dibuat\noleh CRM",
		CreatedBy:     "EMP-1",
		CreatedByName: "Yuswa",
	}.TryPush()
	if err != nil {
		t.Fatalf("TryPush: %v", err)
	}

	if got.Header.Get("X-ASA-Description") != "WO-1 dibuat oleh CRM" {
		t.Fatalf("description = %q", got.Header.Get("X-ASA-Description"))
	}
	if got.Header.Get("X-ASA-Created-By") != "EMP-1" || got.Header.Get("X-ASA-Created-By-Name") != "Yuswa" {
		t.Fatalf("created by = %q %q", got.Header.Get("X-ASA-Created-By"), got.Header.Get("X-ASA-Created-By-Name"))
	}
}

func TestTryPushOmitsEmptyExecutionMetadata(t *testing.T) {
	var got *http.Request

	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
	})

	if _, err := (Webhook{Path: "ping"}).TryPush(); err != nil {
		t.Fatalf("TryPush: %v", err)
	}

	for _, name := range []string{"X-ASA-Description", "X-ASA-Created-By", "X-ASA-Created-By-Name"} {
		if _, ok := got.Header[name]; ok {
			t.Fatalf("header %s should not be sent", name)
		}
	}
}

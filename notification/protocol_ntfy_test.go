package notification

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNtfySendSuccess(t *testing.T) {
	const (
		expectedTitle = "Service Down"
		expectedBody  = "database is unreachable"
		expectedToken = "token-123"
	)

	var gotBody string
	roundTripper := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/alerts" {
			t.Errorf("expected path /alerts, got %s", r.URL.Path)
		}
		if r.Header.Get("Title") != expectedTitle {
			t.Errorf("expected Title header %q, got %q", expectedTitle, r.Header.Get("Title"))
		}
		if r.Header.Get("Authorization") != "Bearer "+expectedToken {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	cfg := &NtfyConfig{
		Server: "https://ntfy.example",
		Topic:  "alerts",
		token:  expectedToken,
	}

	notifier := newNtfyNotifier(cfg)
	notifier.client = &http.Client{Transport: roundTripper}
	err := notifier.send(SendData{
		Title: expectedTitle,
		Body:  expectedBody,
	})
	if err != nil {
		t.Fatalf("expected send to succeed, got error: %v", err)
	}
	if strings.TrimSpace(gotBody) != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, gotBody)
	}
}

func TestNtfySendFailureStatus(t *testing.T) {
	roundTripper := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("nope")),
			Header:     make(http.Header),
		}, nil
	})

	cfg := &NtfyConfig{
		Server: "https://ntfy.example",
		Topic:  "alerts",
	}

	notifier := newNtfyNotifier(cfg)
	notifier.client = &http.Client{Transport: roundTripper}
	if err := notifier.send(SendData{Title: "oops", Body: "fail"}); err == nil {
		t.Fatalf("expected send to return error for non-2xx response")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

package notification

import (
	"errors"
	"strings"
	"testing"
)

type recordingProtocol struct {
	calls []SendData
	err   error
}

func (r *recordingProtocol) send(data SendData) error {
	r.calls = append(r.calls, data)
	return r.err
}

func TestSendWithFallbackOnFailure(t *testing.T) {
	primary := &recordingProtocol{err: errors.New("primary failed")}
	fallback := &recordingProtocol{}
	manager := &Manager{
		protocols: map[string]protocolEntry{
			"primary":  {notify: primary},
			"fallback": {notify: fallback},
		},
	}

	err := manager.SendWithFallback(ProtocolTargets{
		Primary:  []string{"primary"},
		Fallback: []string{"fallback"},
	}, SendData{
		Title: "Alert",
		Body:  "Something happened",
	})
	if err == nil {
		t.Fatalf("expected error from primary notifier")
	}
	if !errors.Is(err, ErrNotificationFailed) {
		t.Fatalf("expected ErrNotificationFailed, got %v", err)
	}
	if len(fallback.calls) != 1 {
		t.Fatalf("expected fallback to be called once, got %d", len(fallback.calls))
	}
	fallbackData := fallback.calls[0]
	if !strings.Contains(fallbackData.Body, "primary failed") {
		t.Fatalf("expected fallback body to include primary error, got %q", fallbackData.Body)
	}
	if !strings.Contains(fallbackData.Body, "Something happened") {
		t.Fatalf("expected fallback body to include original body, got %q", fallbackData.Body)
	}
}

func TestSendWithFallbackNoFailure(t *testing.T) {
	primary := &recordingProtocol{}
	fallback := &recordingProtocol{}
	manager := &Manager{
		protocols: map[string]protocolEntry{
			"primary":  {notify: primary},
			"fallback": {notify: fallback},
		},
	}

	if err := manager.SendWithFallback(ProtocolTargets{
		Primary:  []string{"primary"},
		Fallback: []string{"fallback"},
	}, SendData{
		Title: "OK",
		Body:  "All good",
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(fallback.calls) != 0 {
		t.Fatalf("expected fallback not to be called, got %d", len(fallback.calls))
	}
}

func TestSendInvalidProtocolReturnsErrNotificationFailed(t *testing.T) {
	manager := &Manager{
		protocols: map[string]protocolEntry{},
	}

	err := manager.Send([]string{"missing"}, SendData{Title: "Alert", Body: "Body"})
	if err == nil {
		t.Fatalf("expected error for invalid protocol")
	}
	if !errors.Is(err, ErrNotificationFailed) {
		t.Fatalf("expected ErrNotificationFailed, got %v", err)
	}
}

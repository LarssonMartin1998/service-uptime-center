package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var (
	mockConfig = `
[[service_groups]]
name = "production-group"
auth_token = "token-for-group-1"
max_heartbeat_freq = "5m"

  [[service_groups.services]]
  name = "api-server"

  [[service_groups.services]]
  name = "database"

[[service_groups]]
name = "staging-group"
auth_token = "token-for-group-2"  
max_heartbeat_freq = "1h"

  [[service_groups.services]]
  name = "cache-server"
`
	expectedResult = config{
		ServiceGroups: []serviceGroup{
			{
				Name:             "production-group",
				AuthToken:        "token-for-group-1",
				MaxHeartbeatFreq: time.Minute * 5,
				Services: []service{
					{
						Name: "api-server",
					},
					{
						Name: "database",
					},
				},
			},
			{
				Name:             "staging-group",
				AuthToken:        "token-for-group-2",
				MaxHeartbeatFreq: time.Hour * 1,
				Services: []service{
					{
						Name: "cache-server",
					},
				},
			},
		},
	}
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestConfigCreation(t *testing.T) {
	cfg, err := createConfig(mockConfig)
	if err != nil {
		t.Errorf("failed to create config: %v", err)
	}

	if diff := cmp.Diff(expectedResult, cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("mockConfig should be valid", func(t *testing.T) {
		cfg, err := createConfig(mockConfig)
		if err != nil {
			t.Fatalf("mockConfig should parse without error: %v", err)
		}

		err = validateConfig(&cfg)
		if err != nil {
			t.Errorf("mockConfig should validate without error: %v", err)
		}
	})

	baseGroup := serviceGroup{
		Name:             "test-group",
		AuthToken:        "valid-token",
		MaxHeartbeatFreq: time.Minute * 2,
		Services:         []service{{Name: "test-service"}},
	}

	tests := []struct {
		name        string
		config      config
		expectError error
	}{
		{
			name:   "valid config",
			config: config{ServiceGroups: []serviceGroup{baseGroup}},
		},
		{
			name:        "empty service groups",
			config:      config{ServiceGroups: []serviceGroup{}},
			expectError: errNoServiceGroups,
		},
		{
			name: "service group name too short",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             "a",
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         baseGroup.Services,
				},
			}},
			expectError: errInvalidGroupName,
		},
		{
			name: "service group name too long",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         baseGroup.Services,
				},
			}},
			expectError: errInvalidGroupName,
		},
		{
			name: "auth token too long",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        strings.Repeat("a", 256),
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         baseGroup.Services,
				},
			}},
			expectError: errAuthTokenTooLong,
		},
		{
			name: "heartbeat frequency too short",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: time.Second * 30,
					Services:         baseGroup.Services,
				},
			}},
			expectError: errHeartbeatTooShort,
		},
		{
			name: "no services in group",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         []service{},
				},
			}},
			expectError: errNoServices,
		},
		{
			name: "service name too short",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         []service{{Name: "x"}},
				},
			}},
			expectError: errInvalidServiceName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateConfig(&test.config)

			if test.expectError == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %v but got none", test.expectError)
				} else if !errors.Is(err, test.expectError) {
					t.Errorf("expected error %v but got %v", test.expectError, err)
				}
			}
		})
	}
}

func TestMiddlewareMethods(t *testing.T) {
	for _, testCase := range []struct {
		method     string
		middleware middleware
		expected   bool
	}{
		{
			http.MethodPost,
			middlewareMethodGet,
			false,
		},
		{
			http.MethodGet,
			middlewareMethodGet,
			true,
		},
		{
			http.MethodPut,
			middlewareMethodGet,
			false,
		},
		{
			http.MethodDelete,
			middlewareMethodGet,
			false,
		},
		{
			http.MethodPost,
			middlewareMethodPost,
			true,
		},
		{
			http.MethodGet,
			middlewareMethodPost,
			false,
		},
		{
			http.MethodPut,
			middlewareMethodPost,
			false,
		},
		{
			http.MethodDelete,
			middlewareMethodPost,
			false,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(testCase.method, "/test", nil)

		if result := applyMiddleware(w, r, []middleware{testCase.middleware}); result != testCase.expected {
			t.Errorf("applyMiddleware unexpected result. Got: %t, Expected: %t", result, testCase.expected)
			return
		}

		if testCase.expected {
			continue
		}

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("MiddlewareMethod correctly blocked request but did not have code: %d - StatusMethodNotAllowed", http.StatusMethodNotAllowed)
			return
		}

		allow := w.Header().Get("Allow")
		if len(allow) == 0 {
			t.Errorf("MiddlewareMethod correctly blocked request but did not have the Allow header set")
			return
		}

		if allow == testCase.method {
			t.Errorf("MiddlewareMethod correctly blocked request but has the incorrect value set for the Allow header")
			return
		}
	}
}

func TestMiddlewareAuth(t *testing.T) {
	authToken := "test-token-123"
	middlewareAuth := createAuthMiddleware(authToken)
	for _, test := range []struct {
		header         string
		requestToken   string
		expectedCode   int
		expectedResult bool
	}{
		{
			"Authorization",
			"Bearer " + authToken,
			0,
			true,
		},
		{
			"Authorization",
			"Bearer     " + authToken + "     ",
			0,
			true,
		},
		{
			"Authorization",
			"Bearer     " + authToken + "     t",
			http.StatusUnauthorized,
			false,
		},
		{
			"Authorization",
			"Bearer " + "someothertoken",
			http.StatusUnauthorized,
			false,
		},
		{
			"Authorization",
			authToken,
			http.StatusUnauthorized,
			false,
		},
		{
			"authorization",
			"Bearer " + authToken,
			0,
			true,
		},
		{
			"authorization",
			"bearer " + authToken,
			http.StatusUnauthorized,
			false,
		},
		{
			"",
			"",
			http.StatusBadRequest,
			false,
		},
		{
			"Authorization",
			"",
			http.StatusBadRequest,
			false,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/test", nil)
		r.Header.Set(test.header, test.requestToken)

		if result := middlewareAuth(w, r); result != test.expectedResult {
			t.Errorf("Expected value mismatch. Got: %t, Expected: %t when passing: header-'%s' value-'%s'", result, test.expectedResult, test.header, test.requestToken)
			return
		}

		if test.expectedResult {
			continue
		}

		if w.Code != test.expectedCode {
			t.Errorf("Middleware correctly blocked request but with incorrect code. Got: %d, Expected: %d", w.Code, test.expectedCode)
			return
		}

		wwwAuthHeader := w.Header().Get("WWW-Authenticate")
		if len(wwwAuthHeader) == 0 {
			t.Errorf("Middleware correctly blocked request but WWW-Authenticate header missing!")
			return
		}

		expectedWwwAuthHeader := "Bearer"
		if wwwAuthHeader != expectedWwwAuthHeader {
			t.Errorf("Middleware correctly blocked request but WWW-Authenticate header has incorrect value. Got: %s, Expected %s", wwwAuthHeader, expectedWwwAuthHeader)
			return
		}
	}

	disabledMiddlewareAuth := createAuthMiddleware("")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	if !disabledMiddlewareAuth(w, r) {
		t.Errorf("MiddlewareAuth returned false when authToken is not set, this is not expected behavior.")
		return
	}

	r.Header.Set("Authorization", "this doesnt matter, should still go through with empty authToken")
	if !disabledMiddlewareAuth(w, r) {
		t.Errorf("MiddlewareAuth returned false when authToken is not set, this is not expected behavior.")
		return
	}
}

func TestMiddlewareContentType(t *testing.T) {
	for _, test := range []struct {
		header              string
		expectedContentType string
		requestContentType  string
		code                int
		expected            bool
	}{
		{
			"Content-Type",
			"application/json",
			"application/json",
			0,
			true,
		},
		{
			"content-type",
			"application/json",
			"application/json",
			0,
			true,
		},
		{
			"content-type",
			"application/json",
			"   application/json   ; charset=utf-8",
			0,
			true,
		},
		{
			"content-type",
			"application/json",
			"appLICATion/json",
			0,
			true,
		},
		{
			"unsupported header",
			"application/json",
			"application/json",
			http.StatusBadRequest,
			false,
		},
		{
			"Content-Type",
			"application/json",
			"application/zip",
			http.StatusUnsupportedMediaType,
			false,
		},
		{
			"",
			"",
			"",
			http.StatusBadRequest,
			false,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/test", nil)
		r.Header.Set(test.header, test.requestContentType)

		if result := middlewareContentTypeCheck(w, r, test.expectedContentType); result != test.expected {
			t.Errorf("Expected value mismatch. Got: %t, Expected: %t when passing: header-'%s' value-'%s'", result, test.expected, test.header, test.requestContentType)
			return
		}

		if test.expected {
			continue
		}

		if w.Code != test.code {
			t.Errorf("Middleware correctly blocked request but with incorrect code. Got: %d, Expected: %d", w.Code, test.code)
			return
		}
	}
}

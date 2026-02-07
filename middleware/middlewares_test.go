package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareMethods(t *testing.T) {
	for _, testCase := range []struct {
		method     string
		middleware Middleware
		expected   bool
	}{
		{
			http.MethodPost,
			MiddlewareMethodGet,
			false,
		},
		{
			http.MethodGet,
			MiddlewareMethodGet,
			true,
		},
		{
			http.MethodPut,
			MiddlewareMethodGet,
			false,
		},
		{
			http.MethodDelete,
			MiddlewareMethodGet,
			false,
		},
		{
			http.MethodPost,
			MiddlewareMethodPost,
			true,
		},
		{
			http.MethodGet,
			MiddlewareMethodPost,
			false,
		},
		{
			http.MethodPut,
			MiddlewareMethodPost,
			false,
		},
		{
			http.MethodDelete,
			MiddlewareMethodPost,
			false,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(testCase.method, "/test", nil)

		if result := ApplyMiddleware(w, r, []Middleware{testCase.middleware}); result != testCase.expected {
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
	middlewareAuth := CreateAuthMiddleware(authToken)
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

	disabledMiddlewareAuth := CreateAuthMiddleware("")
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

		if result := MiddlewareContentTypeCheck(w, r, test.expectedContentType); result != test.expected {
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

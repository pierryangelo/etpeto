package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/etpeto/etpeto/internal/platform/ratelimit"
)

func countingHandler(reached *int) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		*reached++
		writer.WriteHeader(http.StatusOK)
	})
}

func send(handler http.Handler, method string, path string, address string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.RemoteAddr = address + ":40000"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestThrottleRefusesBeyondTheLimitAndSaysWhenToRetry(t *testing.T) {
	reached := 0
	handler := Throttle(countingHandler(&reached), []ThrottleRule{{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/local/login",
		Rule:   ratelimit.Rule{Limit: 2, Window: time.Minute},
	}}, false)

	for attempt := 0; attempt < 2; attempt++ {
		if code := send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.5").Code; code != http.StatusOK {
			t.Fatalf("attempt %d = %d", attempt, code)
		}
	}
	response := send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.5")

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if reached != 2 {
		t.Fatalf("the handler ran %d times; the refused request should not reach it", reached)
	}
	retry, err := strconv.Atoi(response.Header().Get("Retry-After"))
	if err != nil || retry < 1 {
		t.Fatalf("Retry-After = %q", response.Header().Get("Retry-After"))
	}
	if got := response.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestThrottleCountsEachAddressSeparately(t *testing.T) {
	reached := 0
	handler := Throttle(countingHandler(&reached), []ThrottleRule{{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/local/login",
		Rule:   ratelimit.Rule{Limit: 1, Window: time.Minute},
	}}, false)

	send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.5")
	// One address exhausting the budget must not lock the whole organization out.
	if code := send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.9").Code; code != http.StatusOK {
		t.Fatalf("a second address was refused: %d", code)
	}
}

func TestThrottleLeavesOtherRoutesAlone(t *testing.T) {
	handler := Throttle(countingHandler(new(int)), []ThrottleRule{{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/local/login",
		Rule:   ratelimit.Rule{Limit: 1, Window: time.Minute},
	}}, false)

	send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.5")

	for _, request := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/documents"},
		{http.MethodPost, "/api/v1/documents"},
		{http.MethodGet, "/api/v1/auth/local/login"},
	} {
		if code := send(handler, request.method, request.path, "203.0.113.5").Code; code != http.StatusOK {
			t.Fatalf("%s %s was throttled: %d", request.method, request.path, code)
		}
	}
}

func TestThrottleMatchesARouteWithIdentifiersInThePath(t *testing.T) {
	handler := Throttle(countingHandler(new(int)), []ThrottleRule{{
		Method: http.MethodPost,
		Path:   "/api/v1/documents/*",
		Rule:   ratelimit.Rule{Limit: 1, Window: time.Minute},
	}}, false)

	send(handler, http.MethodPost, "/api/v1/documents/abc/preview", "203.0.113.5")
	response := send(handler, http.MethodPost, "/api/v1/documents/def/preview", "203.0.113.5")

	// The budget belongs to the route, not to one document: otherwise a client
	// would get a fresh allowance per identifier it invents.
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}

func TestThrottleUsesTheForwardedAddressOnlyWhenTheProxyIsTrusted(t *testing.T) {
	build := func(trusted bool) http.Handler {
		return Throttle(countingHandler(new(int)), []ThrottleRule{{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/local/login",
			Rule:   ratelimit.Rule{Limit: 1, Window: time.Minute},
		}}, trusted)
	}
	forwarded := func(handler http.Handler, address string) int {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/local/login", nil)
		request.RemoteAddr = "10.0.0.2:40000"
		request.Header.Set("X-Forwarded-For", address+", 10.0.0.1")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}

	trusted := build(true)
	forwarded(trusted, "203.0.113.5")
	if code := forwarded(trusted, "203.0.113.9"); code != http.StatusOK {
		t.Fatalf("a different forwarded client was refused: %d", code)
	}
	if code := forwarded(trusted, "203.0.113.5"); code != http.StatusTooManyRequests {
		t.Fatalf("the forwarded client was not throttled: %d", code)
	}

	// Without trust every request behind the same proxy shares one budget, so
	// a spoofed header buys nothing.
	untrusted := build(false)
	forwarded(untrusted, "203.0.113.5")
	if code := forwarded(untrusted, "203.0.113.9"); code != http.StatusTooManyRequests {
		t.Fatalf("a spoofed header bought a fresh allowance: %d", code)
	}
}

func TestThrottleRefusalRevealsNothingAboutTheAccount(t *testing.T) {
	handler := Throttle(countingHandler(new(int)), []ThrottleRule{{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/local/login",
		Rule:   ratelimit.Rule{Limit: 1, Window: time.Minute},
	}}, false)

	send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.5")
	body := send(handler, http.MethodPost, "/api/v1/auth/local/login", "203.0.113.5").Body.String()

	for _, leak := range []string{"@", "usuário", "conta", "senha"} {
		if strings.Contains(body, leak) {
			t.Fatalf("the refusal body mentions %q: %s", leak, body)
		}
	}
}

func TestDefaultThrottlesCoverThePublicAndExpensiveRoutes(t *testing.T) {
	covered := map[string]bool{}
	for _, rule := range DefaultThrottles() {
		covered[rule.Method+" "+rule.Path] = true
		if rule.Rule.Limit <= 0 || rule.Rule.Window <= 0 {
			t.Fatalf("rule %s %s is not configured", rule.Method, rule.Path)
		}
	}
	for _, route := range []string{
		"POST /api/v1/auth/local/login",
		"POST /api/v1/invitations/acceptance",
		"GET /api/v1/invitations",
		"POST /api/v1/documents/*",
	} {
		if !covered[route] {
			t.Fatalf("%s is reachable without a session or is expensive, and is not throttled", route)
		}
	}
}

func TestScaledThrottlesRaiseTheLimitForSharedAddresses(t *testing.T) {
	tight := DefaultThrottles()
	loose := ScaledThrottles(6)

	for index := range tight {
		if loose[index].Rule.Limit != tight[index].Rule.Limit*6 {
			t.Fatalf("%s was not scaled: %d vs %d",
				tight[index].Path, loose[index].Rule.Limit, tight[index].Rule.Limit)
		}
		// The window is the shape of the rule and must not move with the scale.
		if loose[index].Rule.Window != tight[index].Rule.Window {
			t.Fatalf("%s changed its window", tight[index].Path)
		}
	}
}

func TestScaleBelowOneIsTreatedAsOne(t *testing.T) {
	for _, scale := range []int{0, -3} {
		if ScaledThrottles(scale)[0].Rule.Limit != DefaultThrottles()[0].Rule.Limit {
			t.Fatalf("scale %d did not fall back to the default", scale)
		}
	}
}

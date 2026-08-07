package api

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/etpeto/etpeto/internal/platform/ratelimit"
)

// ThrottleRule caps one route. Method and path are matched exactly against what
// the router registered, except for a trailing "*" which matches any suffix.
type ThrottleRule struct {
	Method string
	Path   string
	Rule   ratelimit.Rule
}

// DefaultThrottles protects what is either public or expensive.
//
// The limits assume the deployment the project targets: the Etpeto runs inside
// the organization's own network, where a workstation has its own address. One
// address is therefore close to one person, and a tight limit is precise — a
// legitimate station does not mistype a password twenty times in five minutes.
//
// An installation published to the internet inverts that: remote staff arrive
// through a handful of addresses, and these limits would lock out a department.
// Such an installation should raise them with ETPETO_THROTTLE_SCALE, and lean
// on the per-account limit in the sign-in route, which stays correct either way
// because it counts failures against the account rather than the network.
func DefaultThrottles() []ThrottleRule {
	return ScaledThrottles(1)
}

// ScaledThrottles multiplies every per-address limit, for an installation whose
// clients share addresses. The windows are unchanged.
func ScaledThrottles(scale int) []ThrottleRule {
	if scale < 1 {
		scale = 1
	}
	return []ThrottleRule{
		{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/local/login",
			Rule:   ratelimit.Rule{Limit: 20 * scale, Window: 5 * time.Minute},
		},
		{
			Method: http.MethodPost,
			Path:   "/api/v1/invitations/acceptance",
			Rule:   ratelimit.Rule{Limit: 20 * scale, Window: 5 * time.Minute},
		},
		{
			Method: http.MethodGet,
			Path:   "/api/v1/invitations",
			Rule:   ratelimit.Rule{Limit: 40 * scale, Window: 5 * time.Minute},
		},
		{
			Method: http.MethodPost,
			Path:   "/api/v1/documents/*",
			Rule:   ratelimit.Rule{Limit: 60 * scale, Window: time.Minute},
		},
	}
}

type throttle struct {
	rule    ThrottleRule
	limiter *ratelimit.Limiter
}

// Throttle refuses a client that exceeds a rule, before the request reaches the
// handler. It answers with Problem Details and Retry-After, and says nothing
// about whether the account or invitation exists.
func Throttle(next http.Handler, rules []ThrottleRule, trustProxy bool) http.Handler {
	throttles := make([]throttle, 0, len(rules))
	for _, rule := range rules {
		throttles = append(throttles, throttle{rule: rule, limiter: ratelimit.New(rule.Rule)})
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		for _, candidate := range throttles {
			if !matches(candidate.rule, request) {
				continue
			}
			client := clientAddress(request, trustProxy)
			allowed, wait := candidate.limiter.Allow(candidate.rule.Method + " " + candidate.rule.Path + "|" + client)
			if !allowed {
				// Logged without the path's identifiers, so the log cannot
				// become a list of which invitations were probed.
				slog.Warn("request throttled", "route", candidate.rule.Path, "client", client)
				refuse(writer, wait)
				return
			}
			break
		}
		next.ServeHTTP(writer, request)
	})
}

func matches(rule ThrottleRule, request *http.Request) bool {
	if rule.Method != request.Method {
		return false
	}
	if strings.HasSuffix(rule.Path, "*") {
		return strings.HasPrefix(request.URL.Path, strings.TrimSuffix(rule.Path, "*"))
	}
	return request.URL.Path == rule.Path
}

// clientAddress identifies who is calling.
//
// The API only ever receives traffic through the gateway, which is why the
// forwarded address may be trusted here. Exposing the API directly would make
// that header attacker-controlled, so trust is a deployment decision and not a
// default.
func clientAddress(request *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
			// The left-most entry is the original client; the rest were added
			// by proxies along the way.
			if first, _, found := strings.Cut(forwarded, ","); found {
				return strings.TrimSpace(first)
			}
			return strings.TrimSpace(forwarded)
		}
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return host
}

func refuse(writer http.ResponseWriter, wait time.Duration) {
	seconds := int(wait.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	writer.Header().Set("Retry-After", strconv.Itoa(seconds))
	writer.Header().Set("Content-Type", "application/problem+json")
	writer.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(
		writer,
		`{"title":"Too Many Requests","status":429,"detail":"Muitas tentativas. Tente novamente em %d segundos."}`,
		seconds,
	)
}

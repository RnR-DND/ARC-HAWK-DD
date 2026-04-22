package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() { gin.SetMode(gin.TestMode) }

// stringer implements fmt.Stringer for the toStr/stringerOrFmt tests.
type stringer struct{ s string }

func (s stringer) String() string { return s.s }

func newTestCtx() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/x", nil)
	return c
}

func TestRateLimitKey_TenantPlusUser(t *testing.T) {
	c := newTestCtx()
	tid := uuid.New()
	uid := uuid.New()
	c.Set("tenant_id", tid)
	c.Set("user_id", uid)

	got := rateLimitKey(c)
	wantPrefix := "t:" + tid.String() + "|u:" + uid.String()
	if got != wantPrefix {
		t.Errorf("got %q, want %q", got, wantPrefix)
	}
}

func TestRateLimitKey_TenantOnly(t *testing.T) {
	c := newTestCtx()
	tid := uuid.New()
	c.Set("tenant_id", tid)
	// user_id not set

	got := rateLimitKey(c)
	if !strings.HasPrefix(got, "t:"+tid.String()) {
		t.Errorf("tenant-only key should start with t:<uuid>; got %q", got)
	}
	if strings.Contains(got, "|u:") {
		t.Errorf("tenant-only key should not include |u: segment; got %q", got)
	}
}

func TestRateLimitKey_IPFallback(t *testing.T) {
	c := newTestCtx()
	c.Request.RemoteAddr = "10.1.2.3:4567"

	got := rateLimitKey(c)
	if !strings.HasPrefix(got, "ip:") {
		t.Errorf("unauthenticated request should fall back to ip: prefix; got %q", got)
	}
}

func TestRateLimitKey_StringTenant(t *testing.T) {
	c := newTestCtx()
	c.Set("tenant_id", "string-tenant-123")

	got := rateLimitKey(c)
	if got != "t:string-tenant-123" {
		t.Errorf("got %q, want t:string-tenant-123", got)
	}
}

func TestToStr(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"string", "abc", "abc"},
		{"stringer", stringer{s: "from-stringer"}, "from-stringer"},
		{"uuid", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "00000000-0000-0000-0000-000000000001"},
		{"unknown type", 42, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toStr(tc.in); got != tc.want {
				t.Errorf("toStr(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestStringerOrFmt(t *testing.T) {
	if got := stringerOrFmt(stringer{s: "hello"}); got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
	if got := stringerOrFmt(struct{}{}); got != "" {
		t.Errorf("non-stringer must return empty; got %q", got)
	}
}

func TestRateLimiter_AllowsUpToLimitThenBlocks(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 3, Enabled: true})
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	defer rl.Stop()

	key := "test-bucket"
	for i := 0; i < 3; i++ {
		if !rl.allow(key) {
			t.Fatalf("request %d should be allowed under limit 3", i+1)
		}
	}
	if rl.allow(key) {
		t.Error("4th request should be blocked when limit is 3")
	}
}

func TestRateLimiter_DisabledReturnsNil(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 10, Enabled: false})
	if rl != nil {
		t.Error("disabled rate limiter must return nil so callers can skip it")
	}
}

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/adminauth"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/config"
)

// Test di file ini tidak menyentuh database: semua jalur yang diuji berhenti sebelum memanggil query.

const testPassword = "sandi-uji-unit-123"

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func init() { gin.SetMode(gin.TestMode) }

func testConfig(t *testing.T) config.Config {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return config.Config{
		Env:               "development",
		AdminPasswordHash: hash,
		SessionSecret:     []byte(strings.Repeat("s", 32)),
	}
}

func newUnitRouter(t *testing.T, cfg config.Config, log *slog.Logger) *gin.Engine {
	t.Helper()
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	r, err := NewRouter(cfg, Deps{Log: log, DB: fakePinger{}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func do(r http.Handler, method, path, contentType, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON error: %s", w.Body.String())
	}
	return body.Error.Code
}

func adminCookieFor(cfg config.Config, expires time.Time) *http.Cookie {
	return &http.Cookie{Name: adminCookie, Value: adminauth.Sign(cfg.SessionSecret, expires)}
}

func TestNewRouter_TrustedProxiesTidakValid(t *testing.T) {
	cfg := testConfig(t)
	cfg.TrustedProxies = []string{"172.30.93.0/33"}
	if r, err := NewRouter(cfg, Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}); err == nil || r != nil {
		t.Fatalf("TRUSTED_PROXIES salah ketik diterima, router = %v", r)
	}
}

func TestHealth_PingerPalsu(t *testing.T) {
	cfg := testConfig(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	up, _ := NewRouter(cfg, Deps{Log: log, DB: fakePinger{}})
	w := do(up, http.MethodGet, "/api/health", "", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"database":"up"`) {
		t.Errorf("health up = %d %s", w.Code, w.Body)
	}

	down, _ := NewRouter(cfg, Deps{Log: log, DB: fakePinger{err: errors.New("dial tcp 10.0.0.1:5432: rahasia")}})
	w = do(down, http.MethodGet, "/api/health", "", "")
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"database":"down"`) {
		t.Errorf("health down = %d %s", w.Code, w.Body)
	}
	// Endpoint publik: detail koneksi hanya boleh di log server.
	if strings.Contains(w.Body.String(), "rahasia") || strings.Contains(w.Body.String(), "10.0.0.1") {
		t.Errorf("detail error bocor ke respons: %s", w.Body)
	}
}

func TestNoRoute_JSONDanRequestID(t *testing.T) {
	r := newUnitRouter(t, testConfig(t), nil)
	w1 := do(r, http.MethodGet, "/api/tidak-ada", "", "")
	w2 := do(r, http.MethodGet, "/api/tidak-ada", "", "")
	if w1.Code != http.StatusNotFound || errorCode(t, w1) != CodeNotFound {
		t.Errorf("404 = %d %s", w1.Code, w1.Body)
	}
	id1, id2 := w1.Header().Get("X-Request-ID"), w2.Header().Get("X-Request-ID")
	if id1 == "" || id1 == id2 {
		t.Errorf("X-Request-ID kosong atau kembar: %q %q", id1, id2)
	}
}

func TestRequestID_TidakMemakaiIDDariKlien(t *testing.T) {
	r := newUnitRouter(t, testConfig(t), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("X-Request-ID", "palsu")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-ID") == "palsu" {
		t.Error("request ID dari klien dipakai")
	}
}

// Recovery diuji dengan Deps tanpa Queries: GET /api/event panic karena pointer nil.
func TestRecovery_PanicJadiJSON(t *testing.T) {
	var logBuf bytes.Buffer
	r := newUnitRouter(t, testConfig(t), slog.New(slog.NewJSONHandler(&logBuf, nil)))
	w := do(r, http.MethodGet, "/api/event", "", "")
	if w.Code != http.StatusInternalServerError || errorCode(t, w) != CodeInternal {
		t.Errorf("panic = %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "goroutine") || strings.Contains(w.Body.String(), "nil pointer") {
		t.Errorf("stack trace bocor ke respons: %s", w.Body)
	}
	if !strings.Contains(logBuf.String(), `"msg":"panic"`) {
		t.Error("panic tidak tercatat di log")
	}
}

// NFR-08: log akses tanpa IP, query string, atau body.
func TestAccessLog_TanpaDataPribadi(t *testing.T) {
	var logBuf bytes.Buffer
	r := newUnitRouter(t, testConfig(t), slog.New(slog.NewJSONHandler(&logBuf, nil)))
	req := httptest.NewRequest(http.MethodPost, "/api/registrations?to=Budi+Rahasia",
		strings.NewReader(`{"attendees":[{"full_name":"Sari Pribadi","email":"sari@email.com"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.77:5555"
	r.ServeHTTP(httptest.NewRecorder(), req)

	out := logBuf.String()
	for _, leak := range []string{"Budi", "Sari", "sari@email.com", "203.0.113.77", "to="} {
		if strings.Contains(out, leak) {
			t.Errorf("log memuat %q: %s", leak, out)
		}
	}
	if !strings.Contains(out, `"route":"/api/registrations"`) {
		t.Errorf("route tidak tercatat sebagai pola: %s", out)
	}
}

func TestBukanJSON_415(t *testing.T) {
	cfg := testConfig(t)
	r := newUnitRouter(t, cfg, nil)
	cookie := adminCookieFor(cfg, time.Now().Add(time.Hour))
	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/registrations"},
		{http.MethodPost, "/api/account-check"},
		{http.MethodPost, "/api/admin/login"},
		{http.MethodPost, "/api/admin/vip"},
		{http.MethodPatch, "/api/admin/vip/7b0c2f3e-9a41-4d2b-8c55-0f6e1a2b3c4d"},
	}
	for _, c := range cases {
		for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
			w := do(r, c.method, c.path, ct, `{}`, cookie)
			if w.Code != http.StatusUnsupportedMediaType {
				t.Errorf("%s %s Content-Type %q = %d, mau 415", c.method, c.path, ct, w.Code)
			}
		}
	}
	// Parameter charset tetap dianggap JSON.
	if w := do(r, http.MethodPost, "/api/registrations", "application/json; charset=utf-8", `{`); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("JSON dengan charset = %d", w.Code)
	}
}

func TestPendaftaran_422TanpaDatabase(t *testing.T) {
	r := newUnitRouter(t, testConfig(t), nil)

	w := do(r, http.MethodPost, "/api/registrations", "application/json", `{"attendees":`)
	if w.Code != http.StatusUnprocessableEntity || errorCode(t, w) != CodeValidation {
		t.Errorf("JSON rusak = %d %s", w.Code, w.Body)
	}

	w = do(r, http.MethodPost, "/api/registrations", "application/json", `{"attendees":[{}],"consent":false}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("isian kosong = %d", w.Code)
	}
	var body struct {
		Error struct {
			Fields []FieldError `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	fields := map[string]bool{}
	for _, f := range body.Error.Fields {
		fields[f.Field] = true
	}
	for _, want := range []string{"attendees[0].full_name", "attendees[0].email", "attendees[0].account_name", "consent", "idempotency_key"} {
		if !fields[want] {
			t.Errorf("tidak ada pesan untuk %s: %s", want, w.Body)
		}
	}

	// Body di atas 16 KB tidak dibaca sampai habis.
	big := `{"attendees":[{"full_name":"` + strings.Repeat("a", 17<<10) + `"}]}`
	if w := do(r, http.MethodPost, "/api/registrations", "application/json", big); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("body 17 KB = %d", w.Code)
	}

	w = do(r, http.MethodPost, "/api/account-check", "application/json", `{"account_name":"@@"}`)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "Nama akun wajib diisi.") {
		t.Errorf("account-check @@ = %d %s", w.Code, w.Body)
	}
}

func TestAdmin_TanpaSesi401(t *testing.T) {
	cfg := testConfig(t)
	r := newUnitRouter(t, cfg, nil)
	expired := adminCookieFor(cfg, time.Now().Add(-time.Minute))
	forged := &http.Cookie{Name: adminCookie, Value: adminauth.Sign([]byte(strings.Repeat("x", 32)), time.Now().Add(time.Hour))}
	paths := []struct{ method, path string }{
		{http.MethodGet, "/api/admin/session"},
		{http.MethodGet, "/api/admin/summary"},
		{http.MethodGet, "/api/admin/export?format=csv"},
		{http.MethodGet, "/api/admin/vip"},
		{http.MethodPost, "/api/admin/vip"},
		{http.MethodPatch, "/api/admin/vip/x"},
	}
	for _, p := range paths {
		for name, cookies := range map[string][]*http.Cookie{"tanpa cookie": nil, "kedaluwarsa": {expired}, "secret lain": {forged}} {
			w := do(r, p.method, p.path, "application/json", `{}`, cookies...)
			if w.Code != http.StatusUnauthorized || errorCode(t, w) != CodeUnauthorized {
				t.Errorf("%s %s %s = %d", name, p.method, p.path, w.Code)
			}
		}
	}

	w := do(r, http.MethodGet, "/api/admin/session", "", "", adminCookieFor(cfg, time.Now().Add(time.Hour)))
	if w.Code != http.StatusNoContent || w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("sesi valid = %d, Cache-Control %q", w.Code, w.Header().Get("Cache-Control"))
	}
	if w := do(r, http.MethodGet, "/api/admin/export?format=pdf", "", "", adminCookieFor(cfg, time.Now().Add(time.Hour))); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("format pdf = %d", w.Code)
	}
}

func TestLogin_CookieDanSandi(t *testing.T) {
	cfg := testConfig(t)
	r := newUnitRouter(t, cfg, nil)

	for _, body := range []string{`{"password":"salah"}`, `{"password":""}`, `{}`, `bukan json`} {
		w := do(r, http.MethodPost, "/api/admin/login", "application/json", body)
		if w.Code != http.StatusUnauthorized || errorCode(t, w) != CodeInvalidPassword {
			t.Errorf("%s = %d", body, w.Code)
		}
		if !strings.Contains(w.Body.String(), "Sandi salah. Coba lagi.") {
			t.Errorf("pesan sandi salah = %s", w.Body)
		}
	}

	r = newUnitRouter(t, cfg, nil)
	w := do(r, http.MethodPost, "/api/admin/login", "application/json", `{"password":"`+testPassword+`"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("sandi benar = %d %s", w.Code, w.Body)
	}
	c := w.Result().Cookies()
	if len(c) != 1 {
		t.Fatalf("cookie = %v", c)
	}
	got := c[0]
	if got.Name != adminCookie || !got.HttpOnly || got.SameSite != http.SameSiteStrictMode || got.Path != "/api/admin" ||
		got.MaxAge != int(adminauth.TTL.Seconds()) || got.Secure {
		t.Errorf("atribut cookie salah: %+v", got)
	}
	if w := do(r, http.MethodGet, "/api/admin/session", "", "", got); w.Code != http.StatusNoContent {
		t.Errorf("cookie hasil login ditolak: %d", w.Code)
	}

	cfg.CookieSecure = true
	r = newUnitRouter(t, cfg, nil)
	w = do(r, http.MethodPost, "/api/admin/login", "application/json", `{"password":"`+testPassword+`"}`)
	if c := w.Result().Cookies(); len(c) != 1 || !c[0].Secure {
		t.Errorf("COOKIE_SECURE=true tidak memasang Secure: %v", c)
	}

	w = do(r, http.MethodPost, "/api/admin/logout", "", "")
	if c := w.Result().Cookies(); w.Code != http.StatusNoContent || len(c) != 1 || c[0].MaxAge >= 0 || c[0].Path != "/api/admin" {
		t.Errorf("logout = %d %v", w.Code, c)
	}
}

func TestLogin_RateLimit(t *testing.T) {
	r := newUnitRouter(t, testConfig(t), nil)
	for i := 1; i <= 5; i++ {
		if w := do(r, http.MethodPost, "/api/admin/login", "application/json", `{"password":"salah"}`); w.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan %d = %d, mau 401", i, w.Code)
		}
	}
	w := do(r, http.MethodPost, "/api/admin/login", "application/json", `{"password":"`+testPassword+`"}`)
	if w.Code != http.StatusTooManyRequests || errorCode(t, w) != CodeRateLimited || w.Header().Get("Retry-After") == "" {
		t.Errorf("percobaan ke-6 = %d %s", w.Code, w.Body)
	}
}

func TestIPLimiter(t *testing.T) {
	l := newIPLimiter(time.Hour, 3)
	for i := range 3 {
		if !l.allow("192.0.2.1") {
			t.Fatalf("permintaan %d ditolak sebelum batas", i+1)
		}
	}
	if l.allow("192.0.2.1") {
		t.Error("permintaan ke-4 lolos")
	}
	// Banyak pengguna HP berbagi IP operator, tapi IP lain tetap punya jatah sendiri.
	if !l.allow("192.0.2.2") {
		t.Error("IP lain ikut terkena batas")
	}
}

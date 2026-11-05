//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/testdb"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/vip"
)

// api membungkus router asli dengan database uji. Setiap request memakai IP berbeda, kecuali test rate limit,
// supaya batas per IP tidak ikut menggagalkan test lain.
type api struct {
	t      *testing.T
	env    *testdb.Env
	router *gin.Engine
	reg    *registration.Service
	cookie *http.Cookie
}

var ipSeq atomic.Uint32

func nextIP() string {
	n := ipSeq.Add(1)
	return fmt.Sprintf("10.%d.%d.%d:40000", (n>>16)&0xff, (n>>8)&0xff, n&0xff)
}

func newAPI(t *testing.T) *api {
	t.Helper()
	env := testdb.Setup(t)
	env.Reset(t)
	pool := env.AppPool(t)
	r, err := NewRouter(testConfig(t), Deps{
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:            pool,
		Queries:       sqlc.New(pool),
		Registrations: registration.NewService(pool),
		Vip:           vip.NewService(pool),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &api{t: t, env: env, router: r, reg: registration.NewService(pool)}
}

type reqOpt func(*http.Request)

func fromIP(addr string) reqOpt { return func(r *http.Request) { r.RemoteAddr = addr } }

func (a *api) send(method, path string, body any, opts ...reqOpt) *httptest.ResponseRecorder {
	a.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rd = bytes.NewBufferString(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			a.t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = nextIP()
	if a.cookie != nil {
		req.AddCookie(a.cookie)
	}
	for _, o := range opts {
		o(req)
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	return w
}

func (a *api) login() {
	a.t.Helper()
	w := a.send(http.MethodPost, "/api/admin/login", map[string]string{"password": testPassword})
	if w.Code != http.StatusNoContent {
		a.t.Fatalf("login = %d %s", w.Code, w.Body)
	}
	a.cookie = w.Result().Cookies()[0]
}

type person struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	AccountName string `json:"account_name"`
}

type regBody struct {
	Attendees      []person `json:"attendees"`
	Consent        bool     `json:"consent"`
	IdempotencyKey string   `json:"idempotency_key"`
	Website        string   `json:"website"`
}

type regResult struct {
	Code             string   `json:"code"`
	Type             string   `json:"type"`
	Status           string   `json:"status"`
	WaitlistPosition *int64   `json:"waitlist_position"`
	Attendees        []string `json:"attendees"`
}

type apiError struct {
	Error struct {
		Code    string       `json:"code"`
		Message string       `json:"message"`
		Fields  []FieldError `json:"fields"`
	} `json:"error"`
}

func p(name, account string) person {
	return person{FullName: name, Email: name + "@email.com", AccountName: account}
}

func newReg(people ...person) regBody {
	return regBody{Attendees: people, Consent: true, IdempotencyKey: uuid.NewString()}
}

func (a *api) register(body regBody, opts ...reqOpt) *httptest.ResponseRecorder {
	a.t.Helper()
	return a.send(http.MethodPost, "/api/registrations", body, opts...)
}

// mustRegister untuk menyiapkan keadaan: gagal di sini berarti test-nya sendiri yang salah susun.
func (a *api) mustRegister(people ...person) regResult {
	a.t.Helper()
	w := a.register(newReg(people...))
	if w.Code != http.StatusCreated {
		a.t.Fatalf("pendaftaran %v = %d %s", people, w.Code, w.Body)
	}
	return decode[regResult](a.t, w)
}

func (a *api) addVip(names ...string) {
	a.t.Helper()
	saved := a.cookie
	if saved == nil {
		a.login()
	}
	w := a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": names})
	if w.Code != http.StatusOK {
		a.t.Fatalf("tambah VIP %v = %d %s", names, w.Code, w.Body)
	}
	a.cookie = saved
}

func (a *api) cancel(code string) {
	a.t.Helper()
	if _, err := a.reg.Cancel(a.t.Context(), code); err != nil {
		a.t.Fatalf("batalkan %s: %v", code, err)
	}
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("respons %d bukan JSON yang diharapkan: %s", w.Code, w.Body)
	}
	return v
}

func expectError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) apiError {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, mau %d: %s", w.Code, status, w.Body)
	}
	e := decode[apiError](t, w)
	if e.Error.Code != code {
		t.Fatalf("kode error = %q, mau %q: %s", e.Error.Code, code, w.Body)
	}
	return e
}

func fieldMessage(e apiError, field string) string {
	for _, f := range e.Error.Fields {
		if f.Field == field {
			return f.Message
		}
	}
	return ""
}

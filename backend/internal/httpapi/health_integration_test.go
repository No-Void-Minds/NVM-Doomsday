//go:build integration

package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db"
	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/testdb"
)

// tcpProxy berdiri di antara pool dan tunnel, supaya test bisa memutus database lalu menyambungkannya
// lagi di port yang sama tanpa menyentuh tunnel atau server Postgres.
type tcpProxy struct {
	target string
	addr   string
	mu     sync.Mutex
	ln     net.Listener
	conns  []net.Conn
}

func (p *tcpProxy) start(t *testing.T) {
	t.Helper()
	addr := p.addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	p.mu.Lock()
	p.ln, p.addr = ln, ln.Addr().String()
	p.mu.Unlock()
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				return
			}
			server, err := net.Dial("tcp", p.target)
			if err != nil {
				client.Close()
				continue
			}
			p.mu.Lock()
			p.conns = append(p.conns, client, server)
			p.mu.Unlock()
			go func() { io.Copy(server, client); server.Close() }()
			go func() { io.Copy(client, server); client.Close() }()
		}
	}()
}

func (p *tcpProxy) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ln != nil {
		p.ln.Close()
		p.ln = nil
	}
	for _, c := range p.conns {
		c.Close()
	}
	p.conns = nil
}

func healthStatus(t *testing.T, pinger Pinger) int {
	t.Helper()
	r, err := NewRouter(testConfig(t), Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: pinger})
	if err != nil {
		t.Fatal(err)
	}
	return do(r, http.MethodGet, "/api/health", "", "").Code
}

func TestHealth_503LaluPulihTanpaRestart(t *testing.T) {
	env := testdb.Setup(t)
	cfg, err := pgxpool.ParseConfig(env.AppURL)
	if err != nil {
		t.Fatal("TEST_DATABASE_URL tidak bisa dibaca")
	}
	proxy := &tcpProxy{target: net.JoinHostPort(cfg.ConnConfig.Host, strconv.Itoa(int(cfg.ConnConfig.Port)))}
	proxy.start(t)
	t.Cleanup(proxy.stop)

	host, port, _ := net.SplitHostPort(proxy.addr)
	n, _ := strconv.Atoi(port)
	cfg.ConnConfig.Host, cfg.ConnConfig.Port = host, uint16(n)
	cfg.ConnConfig.ConnectTimeout = time.Second
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Satu router dan satu pool selama test, seperti server yang tidak di-restart.
	r, err := NewRouter(testConfig(t), Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: pool})
	if err != nil {
		t.Fatal(err)
	}
	status := func() int { return do(r, http.MethodGet, "/api/health", "", "").Code }

	if s := status(); s != http.StatusOK {
		t.Fatalf("database tersambung = %d", s)
	}
	proxy.stop()
	if s := status(); s != http.StatusServiceUnavailable {
		t.Errorf("database putus = %d, mau 503", s)
	}
	proxy.start(t)
	var s int
	for range 10 {
		if s = status(); s == http.StatusOK {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if s != http.StatusOK {
		t.Errorf("database tersambung lagi = %d, mau 200 tanpa restart", s)
	}
}

func TestHealth_DatabaseTidakTerjangkau(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://u:p@127.0.0.1:"+freePort(t)+"/x_test?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if s := healthStatus(t, pool); s != http.StatusServiceUnavailable {
		t.Errorf("health = %d, mau 503", s)
	}
}

// Server gagal di awal kalau database tidak terjangkau, daripada jalan tapi semua request error.
func TestNewPool_GagalDiAwal(t *testing.T) {
	start := time.Now()
	pool, err := db.NewPool(context.Background(), "postgres://u:p@127.0.0.1:"+freePort(t)+"/x_test?connect_timeout=1")
	if err == nil {
		pool.Close()
		t.Fatal("pool terbentuk padahal database tidak terjangkau")
	}
	if time.Since(start) > 10*time.Second {
		t.Errorf("gagal setelah %s, terlalu lama", time.Since(start))
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	return port
}

//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// runConcurrently melepas semua goroutine bersamaan, supaya pendaftaran benar-benar berebut kunci event.
func runConcurrently(n int, fn func(i int)) {
	var ready, done sync.WaitGroup
	start := make(chan struct{})
	ready.Add(n)
	done.Add(n)
	for i := range n {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			fn(i)
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
}

func TestBalapan_KapasitasTidakTerlampaui(t *testing.T) {
	a := newAPI(t)
	a.env.Exec(t, `UPDATE event_config SET capacity = 5`)

	const n = 20
	// Respons disimpan dulu dan dibaca setelah semua selesai, karena t.Fatal tidak boleh dipanggil dari goroutine.
	responses := make([]*httptest.ResponseRecorder, n)
	bodies := make([]regBody, n)
	for i := range bodies {
		name := "balap" + uuid.NewString()[:8]
		bodies[i] = newReg(p(name, name))
	}
	runConcurrently(n, func(i int) { responses[i] = a.register(bodies[i]) })
	statuses := make([]int, n)
	results := make([]regResult, n)
	for i, w := range responses {
		statuses[i] = w.Code
		if w.Code == http.StatusCreated {
			results[i] = decode[regResult](t, w)
		}
	}

	var confirmed int
	var positions []int64
	for i, s := range statuses {
		if s != http.StatusCreated {
			t.Fatalf("goroutine %d = %d", i, s)
		}
		switch results[i].Status {
		case "confirmed":
			confirmed++
		case "waitlisted":
			positions = append(positions, *results[i].WaitlistPosition)
		}
	}
	if confirmed > 5 {
		t.Errorf("%d terkonfirmasi, melebihi kapasitas 5", confirmed)
	}
	if confirmed != 5 || len(positions) != 15 {
		t.Errorf("%d terkonfirmasi dan %d daftar tunggu, mau 5 dan 15", confirmed, len(positions))
	}
	// Nomor urut yang ditampilkan di kartu tidak boleh kembar atau bolong.
	sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
	for i, pos := range positions {
		if pos != int64(i+1) {
			t.Errorf("nomor urut daftar tunggu %v, mau 1 sampai 15", positions)
			break
		}
	}
	if db := a.env.QueryInt(t, `SELECT count(*) FROM attendees x JOIN registrations r ON r.id = x.registration_id WHERE r.status = 'confirmed'`); db > 5 {
		t.Errorf("database mencatat %d terkonfirmasi", db)
	}
}

func TestBalapan_NamaVIPSamaHanyaSatuLolos(t *testing.T) {
	a := newAPI(t)
	a.addVip("rebutan")

	const n = 10
	responses := make([]*httptest.ResponseRecorder, n)
	bodies := make([]regBody, n)
	for i := range bodies {
		name := "rebut" + uuid.NewString()[:8]
		bodies[i] = newReg(person{FullName: name, Email: name + "@email.com", AccountName: "@Rebutan"})
	}
	runConcurrently(n, func(i int) { responses[i] = a.register(bodies[i]) })
	statuses := make([]int, n)
	codes := make([]string, n)
	for i, w := range responses {
		statuses[i] = w.Code
		if w.Code == http.StatusConflict {
			codes[i] = decode[apiError](t, w).Error.Code
		}
	}

	created, taken := 0, 0
	for i, s := range statuses {
		switch {
		case s == http.StatusCreated:
			created++
		case s == http.StatusConflict && codes[i] == CodeAccountTaken:
			taken++
		default:
			t.Errorf("goroutine %d = %d %s", i, s, codes[i])
		}
	}
	if created != 1 || taken != n-1 {
		t.Errorf("%d berhasil dan %d account_taken, mau 1 dan %d", created, taken, n-1)
	}
	if db := a.env.QueryInt(t, `SELECT count(*) FROM registrations WHERE vip_account_id IS NOT NULL AND status <> 'cancelled'`); db != 1 {
		t.Errorf("database mencatat %d registrasi untuk nama VIP itu", db)
	}
}

// Dua kiriman dengan key yang sama (misalnya tombol ditekan dua kali) hanya menghasilkan satu registrasi.
func TestBalapan_IdempotencyKeySama(t *testing.T) {
	a := newAPI(t)
	body := newReg(p("ganda", "ganda"))
	const n = 10
	responses := make([]*httptest.ResponseRecorder, n)
	runConcurrently(n, func(i int) { responses[i] = a.register(body) })
	statuses := make([]int, n)
	codes := make([]string, n)
	for i, w := range responses {
		statuses[i] = w.Code
		if w.Code < 300 {
			codes[i] = decode[regResult](t, w).Code
		}
	}
	created := 0
	for i, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusOK:
		default:
			t.Errorf("goroutine %d = %d", i, s)
		}
		if codes[i] != codes[0] {
			t.Errorf("kode berbeda: %v", codes)
			break
		}
	}
	if created != 1 {
		t.Errorf("%d jawaban 201, mau tepat 1", created)
	}
	if db := a.env.QueryInt(t, `SELECT count(*) FROM registrations`); db != 1 {
		t.Errorf("%d registrasi tersimpan", db)
	}
}

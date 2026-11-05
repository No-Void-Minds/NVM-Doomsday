//go:build integration

package httpapi

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type summaryBody struct {
	Capacity               int64 `json:"capacity"`
	ConfirmedAttendees     int64 `json:"confirmed_attendees"`
	WaitlistedAttendees    int64 `json:"waitlisted_attendees"`
	CancelledRegistrations int64 `json:"cancelled_registrations"`
}

func (a *api) summary() summaryBody {
	a.t.Helper()
	w := a.send(http.MethodGet, "/api/admin/summary", nil)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		a.t.Fatalf("summary = %d, Cache-Control %q", w.Code, w.Header().Get("Cache-Control"))
	}
	return decode[summaryBody](a.t, w)
}

func TestAdmin_SummaryCocokDenganDatabase(t *testing.T) {
	a := newAPI(t)
	if w := a.send(http.MethodGet, "/api/admin/summary", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa cookie = %d", w.Code)
	}
	a.login()
	if s := a.summary(); s != (summaryBody{Capacity: 200}) {
		t.Errorf("rekap kosong = %+v", s)
	}

	// Data uji Sprint 7: kapasitas 3, dua VIP, satu batal. Hasil di sana: 2 dari 3, 0, 1 registrasi.
	a.addVip("vipa", "vipb")
	a.env.Exec(t, `UPDATE event_config SET capacity = 3`)
	a.mustRegister(p("a0", "vipa"), p("a1", "x"))
	b := a.mustRegister(p("b0", "vipb"))
	a.cancel(b.Code)
	if s := a.summary(); s != (summaryBody{Capacity: 3, ConfirmedAttendees: 2, WaitlistedAttendees: 0, CancelledRegistrations: 1}) {
		t.Errorf("rekap = %+v", s)
	}

	a.mustRegister(p("c0", "c0"))
	a.mustRegister(p("d0", "d0"))
	got := a.summary()
	want := summaryBody{
		Capacity:               3,
		ConfirmedAttendees:     a.env.QueryInt(t, `SELECT count(*) FROM attendees x JOIN registrations r ON r.id = x.registration_id WHERE r.status = 'confirmed'`),
		WaitlistedAttendees:    a.env.QueryInt(t, `SELECT count(*) FROM attendees x JOIN registrations r ON r.id = x.registration_id WHERE r.status = 'waitlisted'`),
		CancelledRegistrations: a.env.QueryInt(t, `SELECT count(*) FROM registrations WHERE status = 'cancelled'`),
	}
	if got != want || got.ConfirmedAttendees != 3 || got.WaitlistedAttendees != 1 {
		t.Errorf("rekap = %+v, database %+v", got, want)
	}
}

// Kapasitas 2. Urutan daftar: A (terkonfirmasi), B VIP 2 orang (tunggu 1), C (terkonfirmasi), D (tunggu 2),
// E (tunggu 3). Lalu B batal, jadi D dan E maju ke nomor 1 dan 2.
func exportScenario(a *api) {
	a.addVip("vipb")
	a.env.Exec(a.t, `UPDATE event_config SET capacity = 2`)
	a.mustRegister(person{FullName: "=1+1", Email: "a@email.com", AccountName: "@uji2"})
	b := a.mustRegister(p("bayu", "vipb"), p("bella", "x"))
	a.mustRegister(p("citra", "c"))
	a.mustRegister(p("dedi", "d"))
	a.mustRegister(p("eka", "e"))
	a.cancel(b.Code)
}

var fileNamePattern = regexp.MustCompile(`^attachment; filename="nvm-doomsday-peserta-\d{8}-\d{4}\.(csv|xlsx)"$`)

func TestAdmin_EksporCSV(t *testing.T) {
	a := newAPI(t)
	if w := a.send(http.MethodGet, "/api/admin/export?format=csv", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa cookie = %d", w.Code)
	}
	a.login()
	for _, f := range []string{"", "pdf", "CSV"} {
		expectError(t, a.send(http.MethodGet, "/api/admin/export?format="+f, nil), http.StatusUnprocessableEntity, CodeValidation)
	}

	exportScenario(a)
	w := a.send(http.MethodGet, "/api/admin/export?format=csv", nil)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/csv; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("csv = %d, %q, %q", w.Code, w.Header().Get("Content-Type"), w.Header().Get("Cache-Control"))
	}
	if cd := w.Header().Get("Content-Disposition"); !fileNamePattern.MatchString(cd) {
		t.Errorf("Content-Disposition = %q", cd)
	}
	raw := w.Body.Bytes()
	if !bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) || !bytes.Contains(raw, []byte("\r\n")) {
		t.Fatal("tanpa BOM atau CRLF")
	}
	records, err := csv.NewReader(bytes.NewReader(raw[3:])).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	type line struct{ status, pos, role, primary, name, account string }
	want := []line{
		{"Terkonfirmasi", "", "Pendaftar", "'=1+1", "'=1+1", "'@uji2"},
		{"Terkonfirmasi", "", "Pendaftar", "citra", "citra", "c"},
		{"Daftar tunggu", "1", "Pendaftar", "dedi", "dedi", "d"},
		{"Daftar tunggu", "2", "Pendaftar", "eka", "eka", "e"},
		{"Batal", "", "Pendaftar", "bayu", "bayu", "vipb"},
		{"Batal", "", "Tamu", "bayu", "bella", "x"},
	}
	if len(records) != len(want)+1 {
		t.Fatalf("%d baris, mau %d: %v", len(records)-1, len(want), records)
	}
	for i, wl := range want {
		r := records[i+1]
		got := line{r[1], r[2], r[4], r[5], r[6], r[8]}
		if got != wl {
			t.Errorf("baris %d = %+v, mau %+v", i+1, got, wl)
		}
	}
	if records[5][3] != "VIP" || records[1][3] != "Solo" {
		t.Errorf("tipe = %q, %q", records[5][3], records[1][3])
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(records[1][9]) {
		t.Errorf("waktu daftar = %q", records[1][9])
	}
}

func TestAdmin_EksporXLSX(t *testing.T) {
	a := newAPI(t)
	a.login()
	exportScenario(a)
	w := a.send(http.MethodGet, "/api/admin/export?format=xlsx", nil)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("xlsx = %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if cd := w.Header().Get("Content-Disposition"); !fileNamePattern.MatchString(cd) || !strings.HasSuffix(cd, `.xlsx"`) {
		t.Errorf("Content-Disposition = %q", cd)
	}
	f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("Peserta")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 7 {
		t.Fatalf("%d baris, mau 7", len(rows))
	}
	if rows[1][6] != "=1+1" || rows[1][8] != "@uji2" {
		t.Errorf("isian rumus diubah: %q %q", rows[1][6], rows[1][8])
	}
	if formula, _ := f.GetCellFormula("Peserta", "G2"); formula != "" {
		t.Errorf("=1+1 tersimpan sebagai rumus %q", formula)
	}
	if rows[3][2] != "1" || rows[4][2] != "2" {
		t.Errorf("nomor urut = %q, %q", rows[3][2], rows[4][2])
	}
}

func TestAdmin_EksporKosongBerisiJudul(t *testing.T) {
	a := newAPI(t)
	a.login()
	w := a.send(http.MethodGet, "/api/admin/export?format=csv", nil)
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(w.Body.Bytes(), []byte{0xEF, 0xBB, 0xBF}))).ReadAll()
	if err != nil || len(records) != 1 || records[0][0] != "Kode registrasi" {
		t.Errorf("csv kosong = %v %v", records, err)
	}
	w = a.send(http.MethodGet, "/api/admin/export?format=xlsx", nil)
	f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if rows, _ := f.GetRows("Peserta"); len(rows) != 1 {
		t.Errorf("xlsx kosong = %v", rows)
	}
}

type vipList struct {
	Accounts []vipAccountResponse `json:"accounts"`
}

func (a *api) vipList() vipList {
	a.t.Helper()
	w := a.send(http.MethodGet, "/api/admin/vip", nil)
	if w.Code != http.StatusOK {
		a.t.Fatalf("daftar VIP = %d %s", w.Code, w.Body)
	}
	return decode[vipList](a.t, w)
}

func (l vipList) find(name string) *vipAccountResponse {
	for i := range l.Accounts {
		if l.Accounts[i].AccountName == name {
			return &l.Accounts[i]
		}
	}
	return nil
}

func TestAdminVIP_Tanpa401(t *testing.T) {
	a := newAPI(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/vip"},
		{http.MethodPost, "/api/admin/vip"},
		{http.MethodPatch, "/api/admin/vip/" + uuid.NewString()},
	} {
		expectError(t, a.send(c.method, c.path, map[string]any{"names": []string{"budi"}, "is_active": false}), http.StatusUnauthorized, CodeUnauthorized)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM vip_accounts`); n != 0 {
		t.Errorf("nama VIP tersimpan tanpa login")
	}
}

func TestAdminVIP_TambahDanTempel(t *testing.T) {
	a := newAPI(t)
	a.login()
	if l := a.vipList(); len(l.Accounts) != 0 {
		t.Fatalf("daftar awal = %+v", l)
	}

	w := a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": {"satu"}})
	if w.Code != http.StatusOK {
		t.Fatalf("tambah satu = %d %s", w.Code, w.Body)
	}

	// Tempelan 7 baris dari laporan Sprint 9: 3 ditambah, @ dan 51 karakter ditolak, duplikat dan baris kosong dilewati.
	paste := []string{"ujivip1", " @UjiVip2 ", "@", strings.Repeat("a", 51), "UJIVIP1", "", "ujivip3"}
	w = a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": paste})
	if w.Code != http.StatusOK {
		t.Fatalf("tempel = %d %s", w.Code, w.Body)
	}
	rep := decode[vipAddResponse](t, w)
	if strings.Join(rep.Added, ",") != "ujivip1,@UjiVip2,ujivip3" || len(rep.Existing) != 0 || len(rep.Reactivated) != 0 {
		t.Errorf("laporan = %+v", rep)
	}
	if len(rep.Rejected) != 2 || rep.Rejected[0].AccountName != "@" || rep.Rejected[0].Message != "Nama akun wajib diisi." ||
		rep.Rejected[1].Message != "Nama akun maksimal 50 karakter." {
		t.Errorf("ditolak = %+v", rep.Rejected)
	}

	// Nama yang sudah ada dilaporkan, tidak digandakan.
	rep = decode[vipAddResponse](t, a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": {"SATU"}}))
	if strings.Join(rep.Existing, ",") != "satu" {
		t.Errorf("nama yang sudah ada = %+v", rep)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM vip_accounts`); n != 4 {
		t.Errorf("%d nama tersimpan, mau 4", n)
	}
	l := a.vipList()
	acc := l.find("@UjiVip2")
	if acc == nil || !acc.IsActive || acc.RegistrationCode != nil {
		t.Errorf("nama tempelan = %+v", acc)
	}

	for _, body := range []any{map[string][]string{"names": {}}, map[string][]string{"names": {"", "  "}}, map[string]any{}} {
		e := expectError(t, a.send(http.MethodPost, "/api/admin/vip", body), http.StatusUnprocessableEntity, CodeValidation)
		if fieldMessage(e, "names") != "Isi minimal satu nama akun." {
			t.Errorf("%v = %+v", body, e.Error)
		}
	}
	many := make([]string, 501)
	for i := range many {
		many[i] = "nama" + uuid.NewString()[:8]
	}
	e := expectError(t, a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": many}), http.StatusUnprocessableEntity, CodeValidation)
	if fieldMessage(e, "names") != "Maksimal 500 nama sekali tambah. Bagi jadi beberapa kali." {
		t.Errorf("501 nama = %+v", e.Error)
	}
	if w := a.send(http.MethodPost, "/api/admin/vip", "names=budi", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("bukan JSON = %d", w.Code)
	}
}

// Satu baris aneh dari Excel tidak boleh menggagalkan nama lain (keputusan Sprint 9).
func TestAdminVIP_BarisAnehTidakMenggagalkanTempelan(t *testing.T) {
	a := newAPI(t)
	a.login()
	w := a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": {"@ @budi", "sari"}})
	if w.Code != http.StatusOK {
		t.Fatalf("tempelan berisi \"@ @budi\" = %d %s", w.Code, w.Body)
	}
	if a.vipList().find("sari") == nil {
		t.Error("sari tidak ikut tersimpan")
	}
}

func TestAdminVIP_NonaktifkanDanAktifkanLagi(t *testing.T) {
	a := newAPI(t)
	a.login()
	a.addVip("dipakai", "kosong")
	reg := a.mustRegister(p("vina", "dipakai"))
	l := a.vipList()
	used, free := l.find("dipakai"), l.find("kosong")
	if used.RegistrationCode == nil || *used.RegistrationCode != reg.Code || free.RegistrationCode != nil {
		t.Fatalf("status pemakaian = %+v %+v", used, free)
	}

	for _, acc := range []*vipAccountResponse{used, free} {
		if w := a.send(http.MethodPatch, "/api/admin/vip/"+acc.ID, map[string]bool{"is_active": false}); w.Code != http.StatusNoContent {
			t.Fatalf("nonaktifkan %s = %d %s", acc.AccountName, w.Code, w.Body)
		}
	}
	l = a.vipList()
	// Registrasi yang sudah memakai nama tetap VIP dan kodenya tetap terlihat (keputusan Sprint 9).
	if acc := l.find("dipakai"); acc.IsActive || acc.RegistrationCode == nil || *acc.RegistrationCode != reg.Code {
		t.Errorf("nama dipakai setelah nonaktif = %+v", acc)
	}
	if n := a.env.QueryInt(t, `SELECT count(*) FROM registrations WHERE code = $1 AND type = 'vip' AND status = 'confirmed'`, reg.Code); n != 1 {
		t.Error("registrasi ikut berubah saat namanya dinonaktifkan")
	}
	if l.Accounts[len(l.Accounts)-1].IsActive {
		t.Errorf("nama nonaktif tidak di bawah: %+v", l.Accounts)
	}
	if w := a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": "kosong"}); decode[map[string]string](t, w)["type"] != "solo" {
		t.Errorf("nama nonaktif terbaca %s", w.Body)
	}

	// Menambah nama yang dinonaktifkan berarti mengaktifkannya lagi.
	rep := decode[vipAddResponse](t, a.send(http.MethodPost, "/api/admin/vip", map[string][]string{"names": {"KOSONG"}}))
	if strings.Join(rep.Reactivated, ",") != "kosong" {
		t.Errorf("tambah ulang = %+v", rep)
	}
	if w := a.send(http.MethodPatch, "/api/admin/vip/"+used.ID, map[string]bool{"is_active": true}); w.Code != http.StatusNoContent {
		t.Errorf("aktifkan lagi = %d", w.Code)
	}
	if acc := a.vipList().find("dipakai"); !acc.IsActive {
		t.Errorf("nama tidak aktif lagi: %+v", acc)
	}
	if w := a.send(http.MethodPost, "/api/account-check", map[string]string{"account_name": "kosong"}); decode[map[string]string](t, w)["type"] != "vip" {
		t.Errorf("nama yang diaktifkan lagi terbaca %s", w.Body)
	}

	expectError(t, a.send(http.MethodPatch, "/api/admin/vip/bukan-uuid", map[string]bool{"is_active": false}), http.StatusNotFound, CodeNotFound)
	expectError(t, a.send(http.MethodPatch, "/api/admin/vip/"+uuid.NewString(), map[string]bool{"is_active": false}), http.StatusNotFound, CodeNotFound)
	for _, body := range []string{`{}`, `{"is_active":"tidak"}`, `{"is_active":null}`, `bukan json`} {
		e := expectError(t, a.send(http.MethodPatch, "/api/admin/vip/"+used.ID, body), http.StatusUnprocessableEntity, CodeValidation)
		if fieldMessage(e, "is_active") != "Pilih aktif atau nonaktif." {
			t.Errorf("%s = %+v", body, e.Error)
		}
	}
}

func TestAdmin_SesiDanKeluar(t *testing.T) {
	a := newAPI(t)
	a.login()
	if w := a.send(http.MethodGet, "/api/admin/session", nil); w.Code != http.StatusNoContent {
		t.Fatalf("sesi = %d", w.Code)
	}
	w := a.send(http.MethodPost, "/api/admin/logout", nil)
	cleared := w.Result().Cookies()
	if w.Code != http.StatusNoContent || len(cleared) != 1 || cleared[0].Value != "" || cleared[0].MaxAge >= 0 {
		t.Fatalf("logout = %d %+v", w.Code, cleared)
	}
	a.cookie = nil
	expectError(t, a.send(http.MethodGet, "/api/admin/summary", nil), http.StatusUnauthorized, CodeUnauthorized)
}

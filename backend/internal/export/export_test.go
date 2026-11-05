package export

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
)

var t0 = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func dbRow(code string, status sqlc.RegistrationStatus, pos int16, name string, at time.Time) sqlc.ListExportRowsRow {
	role := sqlc.AttendeeRoleGuest
	if pos == 0 {
		role = sqlc.AttendeeRolePrimary
	}
	return sqlc.ListExportRowsRow{
		Code: code, Status: status, Type: sqlc.RegistrationTypeVip, CreatedAt: at,
		Role: role, Position: pos, FullName: name, Email: strings.ToLower(name) + "@email.com",
		AccountName: strings.ToLower(name), PrimaryName: "Utama " + code,
	}
}

// Urutan baris datang dari query ListExportRows. Yang diuji di sini: urutan tidak diubah, dan nomor urut
// daftar tunggu dihitung per registrasi, bukan per peserta, sama seperti query WaitlistPosition.
func sampleRows() []sqlc.ListExportRowsRow {
	c, w, x := sqlc.RegistrationStatusConfirmed, sqlc.RegistrationStatusWaitlisted, sqlc.RegistrationStatusCancelled
	return []sqlc.ListExportRowsRow{
		dbRow("NVM-DD-AAAAAA", c, 0, "Andi", t0),
		dbRow("NVM-DD-AAAAAA", c, 1, "Ayu", t0),
		dbRow("NVM-DD-BBBBBB", w, 0, "Budi", t0.Add(time.Minute)),
		dbRow("NVM-DD-BBBBBB", w, 1, "Bayu", t0.Add(time.Minute)),
		dbRow("NVM-DD-BBBBBB", w, 2, "Bella", t0.Add(time.Minute)),
		dbRow("NVM-DD-CCCCCC", w, 0, "Citra", t0.Add(2*time.Minute)),
		dbRow("NVM-DD-DDDDDD", x, 0, "Dedi", t0.Add(3*time.Minute)),
	}
}

func TestFromDB_UrutanDanNomorUrut(t *testing.T) {
	rows := FromDB(sampleRows())
	want := []struct {
		code, status, role string
		pos                int64
	}{
		{"NVM-DD-AAAAAA", "Terkonfirmasi", "Pendaftar", 0},
		{"NVM-DD-AAAAAA", "Terkonfirmasi", "Tamu", 0},
		{"NVM-DD-BBBBBB", "Daftar tunggu", "Pendaftar", 1},
		{"NVM-DD-BBBBBB", "Daftar tunggu", "Tamu", 1},
		{"NVM-DD-BBBBBB", "Daftar tunggu", "Tamu", 1},
		{"NVM-DD-CCCCCC", "Daftar tunggu", "Pendaftar", 2},
		{"NVM-DD-DDDDDD", "Batal", "Pendaftar", 0},
	}
	if len(rows) != len(want) {
		t.Fatalf("%d baris, mau %d", len(rows), len(want))
	}
	for i, w := range want {
		r := rows[i]
		if r.Code != w.code || r.Status != w.status || r.Role != w.role || r.WaitlistPosition != w.pos {
			t.Errorf("baris %d = %s %s %s %d, mau %+v", i, r.Code, r.Status, r.Role, r.WaitlistPosition, w)
		}
		if r.Type != "VIP" {
			t.Errorf("baris %d tipe %q", i, r.Type)
		}
	}
}

func TestFromDB_Kosong(t *testing.T) {
	if rows := FromDB(nil); rows == nil || len(rows) != 0 {
		t.Errorf("FromDB(nil) = %#v, mau slice kosong", rows)
	}
}

func TestFileName_JamWIB(t *testing.T) {
	// 17.30 UTC sudah lewat tengah malam di WIB, jadi tanggalnya ikut maju.
	now := time.Date(2026, 11, 5, 17, 30, 0, 0, time.UTC)
	if got := FileName(now, "csv"); got != "nvm-doomsday-peserta-20261106-0030.csv" {
		t.Errorf("FileName = %q", got)
	}
	if got := FileName(now, "xlsx"); got != "nvm-doomsday-peserta-20261106-0030.xlsx" {
		t.Errorf("FileName = %q", got)
	}
}

func TestSafeCSVCell(t *testing.T) {
	for _, s := range []string{"=1+1", "+62812", "-2+3", "@SUM(A1)", "\tcmd", "\rcmd"} {
		if got := safeCSVCell(s); got != "'"+s {
			t.Errorf("safeCSVCell(%q) = %q, mau diawali apostrof", s, got)
		}
	}
	for _, s := range []string{"", "Budi", "budi@email.com", "a=1", "'=1", " =1"} {
		if got := safeCSVCell(s); got != s {
			t.Errorf("safeCSVCell(%q) = %q, mau tidak berubah", s, got)
		}
	}
}

func writeCSV(t *testing.T, rows []Row) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteCSV(&buf, rows); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func readCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(b, utf8BOM))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func TestWriteCSV_BOMDanCRLF(t *testing.T) {
	b := writeCSV(t, FromDB(sampleRows()))
	if !bytes.HasPrefix(b, utf8BOM) {
		t.Fatalf("tanpa BOM UTF-8, awal file %x", b[:3])
	}
	body := b[3:]
	lines := bytes.Count(body, []byte("\r\n"))
	if lines != len(sampleRows())+1 {
		t.Errorf("%d baris CRLF, mau %d", lines, len(sampleRows())+1)
	}
	if bytes.Count(body, []byte("\n")) != lines {
		t.Error("ada akhir baris LF tanpa CR")
	}
}

func TestWriteCSV_IsiDanUrutan(t *testing.T) {
	records := readCSV(t, writeCSV(t, FromDB(sampleRows())))
	if strings.Join(records[0], "|") != strings.Join(header, "|") {
		t.Fatalf("judul kolom = %v", records[0])
	}
	codes := make([]string, 0, len(records)-1)
	for _, r := range records[1:] {
		codes = append(codes, r[0]+"/"+r[2])
	}
	want := "NVM-DD-AAAAAA/,NVM-DD-AAAAAA/,NVM-DD-BBBBBB/1,NVM-DD-BBBBBB/1,NVM-DD-BBBBBB/1,NVM-DD-CCCCCC/2,NVM-DD-DDDDDD/"
	if got := strings.Join(codes, ","); got != want {
		t.Errorf("kode/nomor urut = %s\nmau %s", got, want)
	}
	// 10.00 UTC sama dengan 17.00 WIB.
	if records[1][9] != "2026-11-05 17:00:00" {
		t.Errorf("waktu daftar = %q", records[1][9])
	}
}

func TestWriteCSV_FormulaInjection(t *testing.T) {
	rows := []Row{{
		Code: "NVM-DD-AAAAAA", Status: "Terkonfirmasi", Type: "Solo", Role: "Pendaftar",
		PrimaryName: "=1+1", FullName: "+HYPERLINK(\"x\")", Email: "-1@email.com", AccountName: "@uji2",
		RegisteredAt: t0,
	}}
	records := readCSV(t, writeCSV(t, rows))
	got := records[1]
	want := map[int]string{5: "'=1+1", 6: "'+HYPERLINK(\"x\")", 7: "'-1@email.com", 8: "'@uji2"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("kolom %s = %q, mau %q", header[i], got[i], w)
		}
	}

	rows[0].FullName, rows[0].AccountName = "\tTab", "\rCR"
	got = readCSV(t, writeCSV(t, rows))[1]
	// csv.Writer dengan UseCRLF membuang CR tunggal di dalam sel, jadi yang tersisa cukup diawali apostrof.
	if got[6] != "'\tTab" || !strings.HasPrefix(got[8], "'") {
		t.Errorf("tab atau CR di awal sel tidak diberi apostrof: %q, %q", got[6], got[8])
	}
}

func TestWriteCSV_KomaDanKutip(t *testing.T) {
	rows := []Row{{Code: "NVM-DD-AAAAAA", FullName: `Budi, "Bos" Santoso`, RegisteredAt: t0}}
	records := readCSV(t, writeCSV(t, rows))
	if records[1][6] != `Budi, "Bos" Santoso` {
		t.Errorf("nama = %q", records[1][6])
	}
}

func TestWriteCSV_Kosong(t *testing.T) {
	records := readCSV(t, writeCSV(t, nil))
	if len(records) != 1 {
		t.Fatalf("%d baris, mau hanya judul kolom", len(records))
	}
}

func openXLSX(t *testing.T, rows []Row) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, rows); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestWriteXLSX_JudulTebalDanTerkunci(t *testing.T) {
	f := openXLSX(t, FromDB(sampleRows()))
	if sheets := f.GetSheetList(); len(sheets) != 1 || sheets[0] != "Peserta" {
		t.Fatalf("sheet = %v", sheets)
	}
	rows, err := f.GetRows("Peserta")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(rows[0], "|") != strings.Join(header, "|") {
		t.Errorf("judul kolom = %v", rows[0])
	}
	if len(rows) != len(sampleRows())+1 {
		t.Errorf("%d baris, mau %d", len(rows), len(sampleRows())+1)
	}

	styleID, err := f.GetCellStyle("Peserta", "A1")
	if err != nil {
		t.Fatal(err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if style.Font == nil || !style.Font.Bold {
		t.Error("judul kolom tidak tebal")
	}

	panes, err := f.GetPanes("Peserta")
	if err != nil {
		t.Fatal(err)
	}
	if !panes.Freeze || panes.YSplit != 1 {
		t.Errorf("baris judul tidak terkunci: %+v", panes)
	}
}

func TestWriteXLSX_RumusTersimpanSebagaiTeks(t *testing.T) {
	f := openXLSX(t, []Row{{Code: "NVM-DD-AAAAAA", FullName: "=1+1", AccountName: "@uji2", WaitlistPosition: 3, RegisteredAt: t0}})
	formula, err := f.GetCellFormula("Peserta", "G2")
	if err != nil {
		t.Fatal(err)
	}
	if formula != "" {
		t.Errorf("isian =1+1 tersimpan sebagai rumus %q", formula)
	}
	if v, _ := f.GetCellValue("Peserta", "G2"); v != "=1+1" {
		t.Errorf("nama = %q", v)
	}
	if v, _ := f.GetCellValue("Peserta", "I2"); v != "@uji2" {
		t.Errorf("nama akun = %q", v)
	}
	// Nomor urut disimpan sebagai angka supaya bisa diurutkan di Excel.
	if typ, _ := f.GetCellType("Peserta", "C2"); typ != excelize.CellTypeNumber && typ != excelize.CellTypeUnset {
		t.Errorf("nomor urut bertipe %v", typ)
	}
	if v, _ := f.GetCellValue("Peserta", "C2"); v != "3" {
		t.Errorf("nomor urut = %q", v)
	}
}

func TestWriteXLSX_Kosong(t *testing.T) {
	f := openXLSX(t, nil)
	rows, err := f.GetRows("Peserta")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Errorf("%d baris, mau hanya judul kolom", len(rows))
	}
}

package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/db/sqlc"
)

// Indonesia tidak memakai jam musim panas, jadi offset tetap selalu benar, dan tidak bergantung pada
// data zona waktu yang sering tidak ada di image Docker berukuran kecil.
var wib = time.FixedZone("WIB", 7*60*60)

var header = []string{
	"Kode registrasi",
	"Status",
	"Nomor urut daftar tunggu",
	"Tipe",
	"Peran",
	"Nama pendaftar utama",
	"Nama lengkap",
	"Email",
	"Nama akun",
	"Waktu daftar (WIB)",
}

var statusLabel = map[sqlc.RegistrationStatus]string{
	sqlc.RegistrationStatusConfirmed:  "Terkonfirmasi",
	sqlc.RegistrationStatusWaitlisted: "Daftar tunggu",
	sqlc.RegistrationStatusCancelled:  "Batal",
}

var typeLabel = map[sqlc.RegistrationType]string{
	sqlc.RegistrationTypeSolo: "Solo",
	sqlc.RegistrationTypeVip:  "VIP",
}

var roleLabel = map[sqlc.AttendeeRole]string{
	sqlc.AttendeeRolePrimary: "Pendaftar",
	sqlc.AttendeeRoleGuest:   "Tamu",
}

type Row struct {
	Code             string
	Status           string
	WaitlistPosition int64 // 0 berarti bukan daftar tunggu
	Type             string
	Role             string
	PrimaryName      string
	FullName         string
	Email            string
	AccountName      string
	RegisteredAt     time.Time
}

// FromDB mengandalkan urutan dari query ListExportRows: daftar tunggu urut (created_at, id), sama seperti
// query WaitlistPosition, jadi nomor urut di file sama dengan yang ditampilkan di kartu akses.
func FromDB(rows []sqlc.ListExportRowsRow) []Row {
	out := make([]Row, 0, len(rows))
	var pos int64
	var lastCode string
	for _, r := range rows {
		row := Row{
			Code:         r.Code,
			Status:       statusLabel[r.Status],
			Type:         typeLabel[r.Type],
			Role:         roleLabel[r.Role],
			PrimaryName:  r.PrimaryName,
			FullName:     r.FullName,
			Email:        r.Email,
			AccountName:  r.AccountName,
			RegisteredAt: r.CreatedAt,
		}
		if r.Status == sqlc.RegistrationStatusWaitlisted {
			if r.Code != lastCode {
				pos++
			}
			row.WaitlistPosition = pos
		}
		lastCode = r.Code
		out = append(out, row)
	}
	return out
}

func FileName(now time.Time, ext string) string {
	return "nvm-doomsday-peserta-" + now.In(wib).Format("20060102-1504") + "." + ext
}

func (r Row) waitlistCell() string {
	if r.WaitlistPosition == 0 {
		return ""
	}
	return fmt.Sprint(r.WaitlistPosition)
}

func (r Row) registeredCell() string {
	return r.RegisteredAt.In(wib).Format("2006-01-02 15:04:05")
}

func WriteCSV(w io.Writer, rows []Row) error {
	// BOM membuat Excel membaca file sebagai UTF-8, sehingga nama dengan huruf non-ASCII tidak rusak.
	if _, err := io.WriteString(w, "\uFEFF"); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.UseCRLF = true

	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		record := []string{
			r.Code,
			r.Status,
			r.waitlistCell(),
			r.Type,
			r.Role,
			safeCSVCell(r.PrimaryName),
			safeCSVCell(r.FullName),
			safeCSVCell(r.Email),
			safeCSVCell(r.AccountName),
			r.registeredCell(),
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// Isian pendaftar yang diawali karakter ini bisa dijalankan Excel sebagai rumus saat CSV dibuka
// (formula injection). Awalan apostrof membuat Excel memperlakukannya sebagai teks biasa.
func safeCSVCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func WriteXLSX(w io.Writer, rows []Row) error {
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Peserta"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return err
	}

	headerRow := make([]any, len(header))
	for i, h := range header {
		headerRow[i] = h
	}
	if err := f.SetSheetRow(sheet, "A1", &headerRow); err != nil {
		return err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	if err := f.SetRowStyle(sheet, 1, 1, bold); err != nil {
		return err
	}

	// excelize menyimpan string sebagai teks, bukan rumus, jadi isian berawalan "=" tidak dijalankan
	// dan tidak perlu diberi apostrof seperti di CSV.
	for i, r := range rows {
		var waitlist any
		if r.WaitlistPosition > 0 {
			waitlist = r.WaitlistPosition
		}
		cell, err := excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return err
		}
		values := []any{
			r.Code, r.Status, waitlist, r.Type, r.Role,
			r.PrimaryName, r.FullName, r.Email, r.AccountName, r.registeredCell(),
		}
		if err := f.SetSheetRow(sheet, cell, &values); err != nil {
			return err
		}
	}

	// Baris judul tetap terlihat saat admin menggulir ke bawah.
	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft",
	}); err != nil {
		return err
	}
	widths := []float64{18, 15, 24, 8, 11, 26, 26, 30, 20, 20}
	for i, width := range widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, col, col, width); err != nil {
			return err
		}
	}

	return f.Write(w)
}

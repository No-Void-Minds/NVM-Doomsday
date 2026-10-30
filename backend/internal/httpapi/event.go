package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type eventResponse struct {
	Title               string     `json:"title"`
	FilmTitle           string     `json:"film_title"`
	CinemaName          *string    `json:"cinema_name"`
	City                *string    `json:"city"`
	Studio              *string    `json:"studio"`
	GatherAt            *time.Time `json:"gather_at"`
	ShowAt              *time.Time `json:"show_at"`
	MapURL              *string    `json:"map_url"`
	Capacity            int32      `json:"capacity"`
	MaxGuestsVIP        int16      `json:"max_guests_vip"`
	IsOpen              bool       `json:"is_open"`
	SeatsLeft           int64      `json:"seats_left"`
	WaitlistedAttendees int64      `json:"waitlisted_attendees"`
}

func eventHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		ev, err := deps.Queries.GetEvent(ctx)
		if err != nil {
			internalError(c, deps.Log, "ambil event_config", err)
			return
		}
		sum, err := deps.Queries.CountSummary(ctx)
		if err != nil {
			internalError(c, deps.Log, "hitung ringkasan peserta", err)
			return
		}

		seatsLeft := int64(ev.Capacity) - sum.ConfirmedAttendees
		// Admin bisa menurunkan kapasitas di bawah jumlah yang sudah terkonfirmasi lewat CLI.
		// Angka negatif tidak bermakna bagi pengunjung, jadi ditampilkan 0.
		if seatsLeft < 0 {
			seatsLeft = 0
		}

		// Sisa kursi berubah setiap ada pendaftaran, jadi browser dan proxy tidak boleh menyimpan salinan.
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, eventResponse{
			Title:               ev.Title,
			FilmTitle:           ev.FilmTitle,
			CinemaName:          ev.CinemaName,
			City:                ev.City,
			Studio:              ev.Studio,
			GatherAt:            ev.GatherAt,
			ShowAt:              ev.ShowAt,
			MapURL:              ev.MapUrl,
			Capacity:            ev.Capacity,
			MaxGuestsVIP:        ev.MaxGuestsVip,
			IsOpen:              ev.IsOpen,
			SeatsLeft:           seatsLeft,
			WaitlistedAttendees: sum.WaitlistedAttendees,
		})
	}
}

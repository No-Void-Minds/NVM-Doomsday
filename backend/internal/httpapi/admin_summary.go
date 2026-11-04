package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type summaryResponse struct {
	Capacity               int32 `json:"capacity"`
	ConfirmedAttendees     int64 `json:"confirmed_attendees"`
	WaitlistedAttendees    int64 `json:"waitlisted_attendees"`
	CancelledRegistrations int64 `json:"cancelled_registrations"`
}

func summaryHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		ev, err := deps.Queries.GetEvent(ctx)
		if err != nil {
			internalError(c, deps.Log, "ambil event_config", err)
			return
		}
		sum, err := deps.Queries.CountSummary(ctx)
		if err != nil {
			internalError(c, deps.Log, "hitung ringkasan admin", err)
			return
		}

		c.JSON(http.StatusOK, summaryResponse{
			Capacity:               ev.Capacity,
			ConfirmedAttendees:     sum.ConfirmedAttendees,
			WaitlistedAttendees:    sum.WaitlistedAttendees,
			CancelledRegistrations: sum.CancelledRegistrations,
		})
	}
}

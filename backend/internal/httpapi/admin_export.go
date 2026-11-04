package httpapi

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/export"
)

func exportHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		format := c.Query("format")
		if format != "csv" && format != "xlsx" {
			writeError(c, http.StatusUnprocessableEntity, CodeValidation, "Format ekspor harus csv atau xlsx.", nil)
			return
		}

		dbRows, err := deps.Queries.ListExportRows(c.Request.Context())
		if err != nil {
			internalError(c, deps.Log, "ambil data ekspor", err)
			return
		}
		rows := export.FromDB(dbRows)

		// Ditulis ke memori dulu, supaya kegagalan di tengah menghasilkan error yang jelas, bukan file setengah jadi.
		var buf bytes.Buffer
		contentType := "text/csv; charset=utf-8"
		if format == "csv" {
			err = export.WriteCSV(&buf, rows)
		} else {
			contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			err = export.WriteXLSX(&buf, rows)
		}
		if err != nil {
			internalError(c, deps.Log, "tulis file ekspor", err)
			return
		}

		deps.Log.Info("ekspor peserta",
			"request_id", c.GetString(requestIDKey), "format", format, "rows", len(rows))
		c.Header("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, export.FileName(time.Now(), format)))
		c.Data(http.StatusOK, contentType, buf.Bytes())
	}
}

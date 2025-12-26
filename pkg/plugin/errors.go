package plugin

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func sendErrorResponse(w http.ResponseWriter, logMsg string, err error, statusCode int) {
	backend.Logger.Error(logMsg, "error", err)

	var clientMsg string
	switch statusCode {
	case http.StatusBadRequest:
		clientMsg = "invalid request"
	case http.StatusUnauthorized:
		clientMsg = "authentication required"
	case http.StatusForbidden:
		clientMsg = "access denied"
	case http.StatusNotFound:
		clientMsg = "resource not found"
	case http.StatusInternalServerError:
		clientMsg = "internal server error"
	default:
		clientMsg = "request failed"
	}

	http.Error(w, clientMsg, statusCode)
}

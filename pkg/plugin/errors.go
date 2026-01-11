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
		clientMsg = logMsg // Use specific message for bad requests
	case http.StatusUnauthorized:
		clientMsg = "authentication required"
	case http.StatusForbidden:
		clientMsg = logMsg // Use specific message for forbidden
	case http.StatusNotFound:
		clientMsg = logMsg // Use specific message for not found
	case http.StatusTooManyRequests:
		clientMsg = logMsg // Use specific message for rate limits
	case http.StatusNotImplemented:
		clientMsg = logMsg // Use specific message for not implemented
	case http.StatusInternalServerError:
		clientMsg = "internal server error"
	default:
		clientMsg = "request failed"
	}

	http.Error(w, clientMsg, statusCode)
}

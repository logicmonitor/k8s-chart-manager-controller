package healthz

import (
	"net/http"
)

// HandleFunc is an http handler function to expose health metrics.
func HandleFunc(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

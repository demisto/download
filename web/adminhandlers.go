package web

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// downloadLogHandler returns the list of downloads
func (ac *AppContext) downloadLogHandler(w http.ResponseWriter, r *http.Request) {
	l, err := ac.r.ListDownloadLog()
	if err != nil {
		log.WithError(err).Warn("Unable to retrieve download log")
		panic(err)
	}
	writeJSON(w, l)
}

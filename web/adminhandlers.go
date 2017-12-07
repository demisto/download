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

// listDownloadsHandler returns the list of available downloads and versions
func (ac *AppContext) listDownloadsHandler(w http.ResponseWriter, r *http.Request) {
	d, err := ac.r.Downloads()
	if err != nil {
		log.WithError(err).Warn("Unable to retrieve downloads")
		panic(err)
	}
	writeJSON(w, d)
}

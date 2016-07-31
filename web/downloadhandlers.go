package web

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/gorilla/context"
)

// downloadHandler returns the install file
func (ac *AppContext) downloadHandler(w http.ResponseWriter, r *http.Request) {
	u := context.Get(r, "user").(*domain.User)
	token, err := ac.r.Token(u.Token)
	if err != nil {
		log.WithError(err).Errorf("Something is really weird - no token for %#v", u)
		WriteError(w, ErrInternalServer)
		return
	}
	// Token all used up
	if token.Downloads < 1 {
		WriteError(w, &Error{ID: "bad_request", Status: 400, Title: "Invalid Token", Detail: "Token is fully used and no longer allowed to download"})
		return
	}
	d, err := ac.r.Download("free")
	absFile, err := filepath.Abs(d.Path)
	if err != nil {
		log.WithError(err).Errorf("Something wrong with the file path - %#v", d)
		WriteError(w, ErrInternalServer)
		return
	}
	name, dir := filepath.Split(absFile)
	r.URL.Path = name
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	fileServer := http.FileServer(http.Dir(dir))
	fileServer.ServeHTTP(w, r)
}

// uploadHandler allows an admin to upload a new file
func (ac *AppContext) uploadHandler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		log.WithError(err).Error("Failed getting file from request")
		WriteError(w, ErrInternalServer)
		return
	}
	defer file.Close()
	downloadName := r.FormValue("name")
	fileName := r.FormValue("fileName")
	filename := header.Filename
	if len(fileName) > 0 {
		filename = fileName // override with the provided file name
	}
	// Just to be on the safe side
	finalFileName := filepath.Base(filename)
	if finalFileName == "." || finalFileName == "/" {
		log.Errorf("Received weird file name - %s", filename)
		WriteError(w, ErrBadRequest)
		return
	}
	out, err := os.Create(filepath.Join(conf.Options.Dir, finalFileName))
	if err != nil {
		log.WithError(err).Error("Failed saving upload file")
		WriteError(w, ErrInternalServer)
		return
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		log.WithError(err).Error("Failed copying upload file")
		WriteError(w, ErrInternalServer)
		return
	}
	err = ac.r.SetDownload(&domain.Download{Name: downloadName, Path: finalFileName})
	if err != nil {
		log.WithError(err).Error("Error saving download to DB")
		WriteError(w, ErrInternalServer)
		return
	}
	writeJSON(w, map[string]bool{"result": true})
}

package web

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/gorilla/context"
)

// doCheckDownload is the common function between the cookie and parameters check
func (ac *AppContext) doCheckDownload(u *domain.User, w http.ResponseWriter, r *http.Request) {
	if u.Type == domain.UserTypeUser {
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
	}
	writeJSON(w, map[string]bool{"result": true})
}

// checkDownloadHandler checks if the download cookie is valid
func (ac *AppContext) checkDownloadHandler(w http.ResponseWriter, r *http.Request) {
	u := context.Get(r, "user").(*domain.User)
	ac.doCheckDownload(u, w, r)
}

// checkDownloadParamsHandler checks if the download parameters are valid
func (ac *AppContext) checkDownloadParamsHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	token := r.FormValue("token")
	if email == "" || token == "" {
		WriteError(w, ErrMissingPartRequest)
		return
	}
	u, err := ac.r.User(token + "*-*" + email)
	if err != nil {
		log.WithError(err).Errorf("Trying to load user that does not exist for download [%s %s]", token, email)
		WriteError(w, ErrAuth)
		return
	}
	ac.doCheckDownload(u, w, r)
}

// doDownload handles the actual download with either cookie or params
func (ac *AppContext) doDownload(u *domain.User, w http.ResponseWriter, r *http.Request) {
	var token *domain.Token
	if u.Type == domain.UserTypeUser {
		var err error
		token, err = ac.r.Token(u.Token)
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
	}
	downloadName := "free"
	if r.FormValue("ova") != "" {
		downloadName = "ova"
	} else if r.FormValue("ovf") != "" {
		downloadName = "ovf"
	} else if r.FormValue("downloadName") != "" {
		downloadName = r.FormValue("downloadName")
	}

	d, err := ac.r.Download(downloadName)
	if err != nil {
		log.WithError(err).Errorf("Unable to load download %s", downloadName)
		WriteError(w, ErrInternalServer)
		return
	}
	absFile, err := filepath.Abs(d.Path)
	if err != nil {
		log.WithError(err).Errorf("Something wrong with the file path - %#v", d)
		WriteError(w, ErrInternalServer)
		return
	}
	dir, name := filepath.Split(absFile)
	log.Infof("Downloading file %s from %s", name, dir)
	r.URL.Path = name
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	fileServer := http.FileServer(http.Dir(dir))
	fileServer.ServeHTTP(w, r)
	if token != nil {
		token.Downloads--
		err = ac.r.SetToken(token)
		if err != nil {
			log.WithError(err).Errorf("Could not update token in the database - %#v", token)
		}
	}
	// Just log the download
	err = ac.r.LogDownload(u, d, r.RemoteAddr)
	if err != nil {
		log.WithError(err).Errorf("Could not log the download in the database - %#v [%v]", u, d)
	}
}

// downloadHandler returns the install file
func (ac *AppContext) downloadHandler(w http.ResponseWriter, r *http.Request) {
	u := context.Get(r, "user").(*domain.User)
	ac.doDownload(u, w, r)
}

// downloadParamsHandler returns the install file using parameters
func (ac *AppContext) downloadParamsHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	token := r.FormValue("token")
	if email == "" || token == "" {
		WriteError(w, ErrMissingPartRequest)
		return
	}
	u, err := ac.r.User(token + "*-*" + email)
	if err != nil {
		log.WithError(err).Errorf("Trying to load user that does not exist for download [%s %s]", token, email)
		WriteError(w, ErrAuth)
		return
	}
	ac.doDownload(u, w, r)
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
	gitHash := r.FormValue("gitHash")
	if gitHash == "" {
		gitHash = "N/A"
	}
	// Just to be on the safe side
	finalFileName := filepath.Base(filename)
	if finalFileName == "." || finalFileName == "/" {
		log.Errorf("Received weird file name - %s", filename)
		WriteError(w, ErrBadRequest)
		return
	}
	finalPath := filepath.Join(conf.Options.Dir, finalFileName)
	out, err := os.Create(finalPath)
	if err != nil {
		log.WithError(err).Error("Failed saving upload file - %s", finalPath)
		WriteError(w, ErrInternalServer)
		return
	}
	defer out.Close()
	h := sha256.New()
	tee := io.MultiWriter(h, out)
	_, err = io.Copy(tee, file)
	if err != nil {
		log.WithError(err).Error("Failed copying upload file")
		WriteError(w, ErrInternalServer)
		return
	}
	err = ac.r.SetDownload(&domain.Download{Name: downloadName, Path: finalPath, SHA256: base64.StdEncoding.EncodeToString(h.Sum(nil)), GitHash: gitHash})
	if err != nil {
		log.WithError(err).Error("Error saving download to DB")
		WriteError(w, ErrInternalServer)
		return
	}
	writeJSON(w, map[string]bool{"result": true})
}

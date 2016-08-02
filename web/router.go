package web

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/demisto/download/util"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

// Main handlers
var public string

// ServeGzipFiles ...
func (r *Router) ServeGzipFiles(path string, root http.FileSystem) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	r.GET(path, func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		req.URL.Path = ps.ByName("filepath")
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") &&
			(strings.HasSuffix(req.URL.Path, ".js") ||
				strings.HasSuffix(req.URL.Path, ".html") ||
				strings.HasSuffix(req.URL.Path, ".css")) {
			w.Header().Set("Content-Encoding", "gzip")
		}

		fileServer := http.FileServer(root)
		fileServer.ServeHTTP(w, req)
	})
}

func pageHandler(file string) func(w http.ResponseWriter, r *http.Request) {
	m := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, public+file)
	}

	return m
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeWithFilter(w http.ResponseWriter, v interface{}, filters ...string) {
	log.Debugf("Filters in web level : %q", filters)
	b, err := util.MarshalWithFilter(v, filters...)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// Router

// Router handles the web requests routing
type Router struct {
	*httprouter.Router
	staticHandlers alice.Chain
	commonHandlers alice.Chain
	authHandlers   alice.Chain
	fileHandlers   alice.Chain
	appContext     *AppContext
}

// Get handles GET requests
func (r *Router) Get(path string, requires []domain.UserType, handler http.Handler) {
	r.GET(path, wrapHandler(requires, handler))
}

// Post handles POST requests
func (r *Router) Post(path string, requires []domain.UserType, handler http.Handler) {
	r.POST(path, wrapHandler(requires, handler))
}

// Put handles PUT requests
func (r *Router) Put(path string, requires []domain.UserType, handler http.Handler) {
	r.PUT(path, wrapHandler(requires, handler))
}

// Delete handles DELETE requests
func (r *Router) Delete(path string, requires []domain.UserType, handler http.Handler) {
	r.DELETE(path, wrapHandler(requires, handler))
}

func handlePublicPath(pubPath string) {
	switch {
	// absolute path
	case len(pubPath) > 1 && (pubPath[0] == '/' || pubPath[0] == '\\'):
		public = pubPath
	// absolute path win
	case len(pubPath) > 2 && pubPath[1] == ':':
		public = pubPath
	// relative
	case len(pubPath) > 1 && pubPath[0] == '.':
		public = pubPath
	default:
		public = "./" + pubPath
	}
	if public[len(public)-1] != '/' && public[len(public)-1] != '\\' {
		public = fmt.Sprintf("%s%c", public, os.PathSeparator)
	}
	log.Infof("Using public path %v", public)
}

// New creates a new router
func New(appC *AppContext, pubPath string) *Router {
	initBruteForceMap(false)
	handlePublicPath(pubPath)
	r := &Router{Router: httprouter.New()}
	r.appContext = appC
	r.staticHandlers = alice.New(context.ClearHandler, loggingHandler, csrfHandler, recoverHandler, clickjackingHandler)
	r.commonHandlers = r.staticHandlers.Append(acceptHandler)
	r.authHandlers = r.commonHandlers.Append(appC.authHandler, appC.permissionsHandler)
	r.fileHandlers = r.staticHandlers.Append(appC.authHandler, appC.permissionsHandler)
	r.registerStaticHandlers()
	r.registerApplicationHandlers()
	return r
}

// Static handlers
func (r *Router) registerStaticHandlers() {
	// 404 not found handler
	r.NotFound = r.staticHandlers.ThenFunc(notFoundHandler)

	// Static
	r.Get("/", nil, r.staticHandlers.ThenFunc(pageHandler("index.html")))
	r.Get("/favicon.ico", nil, r.staticHandlers.ThenFunc(pageHandler("favicon.ico")))
	r.Get("/style.css", nil, r.staticHandlers.ThenFunc(pageHandler("style.css")))
	r.Get("/404", nil, r.staticHandlers.ThenFunc(pageHandler("404.html")))
	r.Get("/demisto-free-edition", nil, r.staticHandlers.ThenFunc(pageHandler("download.html")))
	r.Get("/free-edition-install-guide", nil, r.staticHandlers.ThenFunc(pageHandler("Demisto Getting Started Guide Standalone.pdf")))
	r.ServeGzipFiles("/assets/*filepath", http.Dir(public+"assets"))
}

// handlers that are available just in stand alone mode and not in proxy mode
func (r *Router) registerApplicationHandlers() {
	// Security
	r.Post("/login", nil, r.commonHandlers.Append(jsonContentTypeHandler, bodyHandler(credentials{})).ThenFunc(r.appContext.loginHandler))
	r.Post("/logout", nil, r.authHandlers.ThenFunc(r.appContext.logoutHandler))
	r.Get("/user", nil, r.authHandlers.ThenFunc(r.appContext.userCurrHandler))
	r.Post("/user", []domain.UserType{domain.UserTypeAdmin}, r.authHandlers.Append(jsonContentTypeHandler, bodyHandler(userDetails{})).ThenFunc(r.appContext.handleUserUpdate))
	// Quiz
	r.Get("/quiz", nil, r.commonHandlers.ThenFunc(r.appContext.quizHandler))
	r.Get("/secret-url-for-you-to-find", nil, r.commonHandlers.ThenFunc(r.appContext.secretURLForAnswersHandler))
	r.Get("/quizall", []domain.UserType{domain.UserTypeAdmin}, r.commonHandlers.ThenFunc(r.appContext.quizAllHandler))
	r.Post("/quiz", []domain.UserType{domain.UserTypeAdmin}, r.authHandlers.Append(jsonContentTypeHandler, bodyHandler(domain.Quiz{})).ThenFunc(r.appContext.updateQuizHandler))
	r.Post("/check", nil, r.commonHandlers.Append(jsonContentTypeHandler, bodyHandler(quizResponse{})).ThenFunc(r.appContext.checkQuiz))
	// Token
	r.Get("/token", []domain.UserType{domain.UserTypeAdmin}, r.authHandlers.ThenFunc(r.appContext.tokenHandler))
	r.Post("/tokens/generate", []domain.UserType{domain.UserTypeAdmin}, r.authHandlers.Append(jsonContentTypeHandler, bodyHandler(newTokens{})).ThenFunc(r.appContext.createTokensHandler))
	r.Post("/token", []domain.UserType{domain.UserTypeAdmin}, r.authHandlers.Append(jsonContentTypeHandler, bodyHandler(domain.Token{})).ThenFunc(r.appContext.updateToken))
	// Downloads
	r.Get("/check-download", []domain.UserType{domain.UserTypeUser, domain.UserTypeAdmin}, r.authHandlers.ThenFunc(r.appContext.checkDownloadHandler))
	r.Get("/download", []domain.UserType{domain.UserTypeUser, domain.UserTypeAdmin}, r.fileHandlers.ThenFunc(r.appContext.downloadHandler))
	r.Post("/upload", []domain.UserType{domain.UserTypeAdmin}, r.authHandlers.Append(multipartContentTypeHandler).ThenFunc(r.appContext.uploadHandler))
}

func wrapHandler(requires []domain.UserType, h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		context.Set(r, "params", ps)
		context.Set(r, "requires", requires)
		h.ServeHTTP(w, r)
	}
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, conf.Options.ExternalAddress+r.RequestURI, http.StatusMovedPermanently)
}

// Serve - creates the relevant listeners
func (r *Router) Serve() {
	var err error
	if conf.Options.SSL.Cert != "" {
		// First, listen on the HTTP address with redirect
		go func() {
			err := http.ListenAndServe(conf.Options.HTTPAddress, http.HandlerFunc(redirectToHTTPS))
			if err != nil {
				log.Fatal(err)
			}
		}()
		addr := conf.Options.Address
		if addr == "" {
			addr = ":https"
		}
		server := &http.Server{Addr: conf.Options.Address, Handler: r}
		config, err := GetTLSConfig()
		if err != nil {
			log.Fatal(err)
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
		err = server.Serve(tlsListener)
	} else {
		err = http.ListenAndServe(conf.Options.Address, r)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// 404 not found handler
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/404", http.StatusSeeOther)
}

// GetTLSConfig ...
func GetTLSConfig() (config *tls.Config, err error) {
	certs := make([]tls.Certificate, 1)
	certs[0], err = tls.X509KeyPair([]byte(conf.Options.SSL.Cert), []byte(conf.Options.SSL.Key))
	if err != nil {
		return nil, err
	}
	config = &tls.Config{
		NextProtos:               []string{"http/1.1"},
		MinVersion:               tls.VersionTLS12,
		Certificates:             certs,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		},
	}
	return
}

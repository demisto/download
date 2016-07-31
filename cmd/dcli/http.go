package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/demisto/download/domain"
)

const (
	// xsrfTokenKey ...
	xsrfTokenKey = "X-XSRF-TOKEN"
	// xsrfCookieKey ...
	xsrfCookieKey = "XSRF-TOKEN"
)

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// Client implements a client for the Demisto download server
type Client struct {
	*http.Client
	credentials *credentials
	username    string
	password    string
	server      string
	token       string
}

// New client that does not do anything yet before the login
func New(username, password, server string, insecure bool) (*Client, error) {
	if username == "" || password == "" || server == "" {
		return nil, fmt.Errorf("Please provide all the parameters")
	}
	if !strings.HasSuffix(server, "/") {
		server += "/"
	}
	cookieJar, _ := cookiejar.New(nil)
	c := &Client{Client: &http.Client{Jar: cookieJar}, credentials: &credentials{User: username, Password: password}, server: server}
	if insecure {
		c.Client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	c.Jar = cookieJar
	req, err := http.NewRequest("GET", server, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	for _, element := range resp.Cookies() {
		if element.Name == xsrfCookieKey {
			c.token = element.Value
		}
	}
	return c, nil
}

// handleError will handle responses with status code different from success
func (c *Client) handleError(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Unexpected status code: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func (c *Client) req(method, path, contentType string, body io.Reader, result interface{}) error {
	req, err := http.NewRequest(method, c.server+path, body)
	if err != nil {
		return err
	}
	req.Header.Add("Accept", "application/json")
	if contentType == "" {
		req.Header.Add("Content-type", "application/json")
	} else {
		req.Header.Add("Content-type", contentType)
	}
	req.Header.Add(xsrfTokenKey, c.token)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err = c.handleError(resp); err != nil {
		return err
	}
	if result != nil {
		switch result := result.(type) {
		// Should we just dump the response body
		case io.Writer:
			if _, err = io.Copy(result, resp.Body); err != nil {
				return err
			}
		default:
			if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
				return err
			}
		}
	}
	return nil
}

// Login to the Demisto download server, and returns statues code
func (c *Client) Login() (*domain.User, error) {
	creds, err := json.Marshal(c.credentials)
	if err != nil {
		return nil, err
	}
	u := &domain.User{}
	err = c.req("POST", "login", "", bytes.NewBuffer(creds), u)
	return u, err
}

// Logout from the Demisto server
func (c *Client) Logout() error {
	return c.req("POST", "logout", "", nil, nil)
}

func (c *Client) Tokens() (tokens []domain.Token, err error) {
	err = c.req("GET", "token", "", nil, &tokens)
	return
}

type userDetails struct {
	Username string          `json:"username"`
	Password string          `json:"password"`
	Email    string          `json:"email"`
	Name     string          `json:"name"`
	Type     domain.UserType `json:"type"`
	Token    string          `json:"token"`
}

func (c *Client) SetUser(u *userDetails) (*domain.User, error) {
	b, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	res := &domain.User{}
	err = c.req("POST", "user", "", bytes.NewBuffer(b), res)
	return res, err
}

package repo

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	username VARCHAR(128) NOT NULL,
	hash VARCHAR(128),
	email VARCHAR(128),
	name VARCHAR(128),
	type INT NOT NULL,
	modify_date TIMESTAMP NOT NULL,
	last_login TIMESTAMP NOT NULL,
	token VARCHAR(128),
	CONSTRAINT users_pk PRIMARY KEY (username)
);
CREATE TABLE IF NOT EXISTS tokens (
	name VARCHAR(30) NOT NULL,
	downloads INT NOT NULL,
	CONSTRAINT tokens_pk PRIMARY KEY (name)
);
CREATE TABLE IF NOT EXISTS questions (
	name VARCHAR(30) NOT NULL,
	question VARCHAR(512) NOT NULL,
	answers VARCHAR(1024) NOT NULL,
	correct VARCHAR(30) NOT NULL,
	CONSTRAINT questions_pk PRIMARY KEY (name)
);
CREATE TABLE IF NOT EXISTS downloads (
	name VARCHAR(30) NOT NULL,
	path VARCHAR(1024) NOT NULL,
	modify_date TIMESTAMP NOT NULL,
	CONSTRAINT download_pk PRIMARY KEY (name)
)`

var (
	// ErrNotFound is a not found error if Get does not retrieve a value
	ErrNotFound = errors.New("not_found")
)

type Repo struct {
	db   *sqlx.DB
	stop chan bool
}

// New repo is returned
// To create the relevant MySQL databases on local please do the following:
//   mysql -u root (if password is set then add -p)
//   mysql> CREATE DATABASE download CHARACTER SET = utf8;
//   mysql> CREATE USER download IDENTIFIED BY 'password';
//   mysql> GRANT ALL on download.* TO download;
//   mysql> drop user ''@'localhost';
// The last command drops the anonymous user
// Repo basically ignores optimistic locking and will have lost update problem but since this is considered
// low volume and not a big deal if we allow additional download - decided to just ignore
func New() (*Repo, error) {
	logrus.Infof("Using MySQL at %s with user %s", conf.Options.DB.ConnectString, conf.Options.DB.Username)
	// If we specified TLS connection, we need the certificate files
	if conf.Options.DB.ServerCA != "" {
		rootCertPool := x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM([]byte(conf.Options.DB.ServerCA)); !ok {
			return nil, errors.New("Unable to add ServerCA PEM")
		}
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.X509KeyPair([]byte(conf.Options.DB.ClientCert), []byte(conf.Options.DB.ClientKey))
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)
		// Make sure to pass &tls=demisto on the connect string to use this configuration
		mysql.RegisterTLSConfig("demisto", &tls.Config{
			RootCAs:            rootCertPool,
			Certificates:       clientCert,
			InsecureSkipVerify: true,
		})
	}
	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@%s", conf.Options.DB.Username, conf.Options.DB.Password, conf.Options.DB.ConnectString))
	if err != nil {
		return nil, err
	}
	logrus.Infof("Connected - %v", time.Now())
	// Have to set it to make sure no connection is left idle and being killed
	db.SetMaxIdleConns(0)
	creates := strings.Split(schema, ";")
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	for _, create := range creates {
		if strings.TrimSpace(create) == "" {
			continue
		}
		_, err = tx.Exec(create)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	logrus.Info("Schema creation is done")
	r := &Repo{
		db:   db,
		stop: make(chan bool, 1),
	}
	return r, nil
}

func (r *Repo) Close() error {
	r.stop <- true
	return r.db.Close()
}

func (r *Repo) get(tableName, field, id string, data interface{}) error {
	err := r.db.Get(data, "SELECT * FROM "+tableName+" WHERE "+field+" = ?", id)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

func (r *Repo) del(tableName, id string) error {
	_, err := r.db.Exec("DELETE FROM "+tableName+" WHERE id = ?", id)
	return err
}

func (r *Repo) User(username string) (*domain.User, error) {
	user := &domain.User{}
	err := r.get("users", "username", username, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repo) SetUser(u *domain.User) error {
	logrus.Infof("Saving user - %s", u.Username)
	if u.ModifyDate.IsZero() {
		u.ModifyDate = time.Now()
	}
	_, err := r.db.Exec(`INSERT INTO users (
username, hash, email, name, type, modify_date, last_login, token)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
hash = ?,
email = ?,
name = ?,
type = ?,
modify_date = ?,
last_login = ?,
token = ?`,
		u.Username, u.Hash, u.Email, u.Name, u.Type, u.ModifyDate, u.LastLogin, u.Token,
		u.Hash, u.Email, u.Name, u.Type, u.ModifyDate, u.LastLogin, u.Token)
	return err
}

func (r *Repo) Questions() (q []domain.Quiz, err error) {
	type question struct {
		Name     string
		Question string
		Answers  string
		Correct  string
	}
	var questions []question
	err = r.db.Select(&questions, "SELECT * FROM questions")
	if err != nil {
		return
	}
	for i := range questions {
		q = append(q, domain.Quiz{
			Name:     questions[i].Name,
			Question: questions[i].Question,
			Answers:  domain.AnswersFromString(questions[i].Answers),
			Correct:  domain.CorrectFromString(questions[i].Correct)})
	}
	return
}

func (r *Repo) SetQuestion(q *domain.Quiz) error {
	logrus.Infof("Saving quiz - %s", q.Name)
	answers := domain.AnswersToString(q.Answers)
	correct := domain.CorrectToString(q.Correct)
	_, err := r.db.Exec(`INSERT INTO questions (name, question, answers, correct) VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
question = ?,
answers = ?,
correct = ?`,
		q.Name, q.Question, answers, correct, q.Question, answers, correct)
	return err
}

func (r *Repo) Token(name string) (*domain.Token, error) {
	token := &domain.Token{}
	err := r.get("tokens", "name", name, token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *Repo) Tokens() (t []domain.Token, err error) {
	err = r.db.Select(&t, "SELECT * FROM tokens")
	return
}

func (r *Repo) OpenTokens() (t []domain.Token, err error) {
	err = r.db.Select(&t, "SELECT * FROM tokens WHERE downloads > 0")
	return
}

func (r *Repo) SetToken(t *domain.Token) error {
	logrus.Infof("Saving token - %s", t.Name)
	_, err := r.db.Exec(`INSERT INTO tokens (name, downloads) VALUES (?, ?) ON DUPLICATE KEY UPDATE downloads = ?`,
		t.Name, t.Downloads, t.Downloads)
	return err
}

func (r *Repo) Download(name string) (*domain.Download, error) {
	d := &domain.Download{}
	err := r.get("downloads", "name", name, d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *Repo) SetDownload(d *domain.Download) error {
	logrus.Infof("Saving download - %#v", d)
	if d.ModifyDate.IsZero() {
		d.ModifyDate = time.Now()
	}
	_, err := r.db.Exec(`INSERT INTO downloads (name, path, modify_date) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE path = ?, modify_date = ?`,
		d.Name, d.Path, d.ModifyDate, d.Path, d.ModifyDate)
	return err
}

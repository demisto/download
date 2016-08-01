// Package conf provides basic configuration handling from a file exposing a single global struct with all configuration.
package conf

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
)

// Options anonymous struct holds the global configuration options for the server
var Options struct {
	// The address to listen on
	Address string
	// The HTTP address to listen on if the main address is HTTPS
	HTTPAddress string
	// ExternalAddress to our web tier
	ExternalAddress string
	// Security defintions
	Security struct {
		// The secret session key that is used to symmetrically encrypt sessions stored in cookies
		SessionKey string
		// Session timeout in minutes
		Timeout int
		// Database encryption key used to encrypt sensitive data
		DBKey string
	}
	// SSL configuration
	SSL struct {
		// The certificate file
		Cert string
		// The private key file
		Key string
	}
	// DB properties
	DB struct {
		// ConnectString how to connect to DB
		ConnectString string
		// Username for the DB
		Username string
		// Password for DB
		Password string
		// ServerCA for TLS
		ServerCA string
		// ClientCert for TLS
		ClientCert string
		// ClientKey for TLS
		ClientKey string
	}
	// Dir where to place the files
	Dir string
	// Location of the static resources
	Static string
}

// The pipe writer to wrap around standard logger. It is configured in main.
var LogWriter *io.PipeWriter

// Load loads configuration from a file.
func Load(filename string) error {
	options, err := ioutil.ReadFile(filename)
	if err != nil {
		logrus.WithField("error", err).Warn("Could not open config file and not using default")
		return err
	} else {
		err = json.Unmarshal(options, &Options)
		if err != nil {
			return err
		}
	}
	if Options.Dir == "" {
		Options.Dir = "."
	}
	finalOptions, err := json.MarshalIndent(&Options, "", "  ")
	if err != nil {
		return err
	}
	logrus.Infof("Using options:\n%s\n", string(finalOptions))
	return nil
}

func Default() {
	Options.Address = ":9090"
	Options.Security.SessionKey = "kukuKiki1234qawsed.Strazaaplokij"
	Options.Security.DBKey = Options.Security.SessionKey
	Options.Security.Timeout = 1440
	Options.DB.Username = "download"
	Options.DB.Password = "demisto1999"
	Options.DB.ConnectString = "tcp/download?parseTime=true"
	Options.Dir = "."
	Options.Static = "static"
}

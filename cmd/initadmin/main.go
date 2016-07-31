package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/demisto/download/repo"
)

var (
	confFile = flag.String("conf", "", "Path to configuration file in JSON format")
	user     = flag.String("u", "admin", "The user to create")
	pass     = flag.String("p", "", "The password to set")
)

func stderr(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	os.Exit(1)
}

func check(e error) {
	if e != nil {
		stderr("Error - %v\n", e)
	}
}

func main() {
	flag.Parse()
	if *pass == "" {
		stderr("Please provide the password")
	}
	conf.Default()
	if *confFile != "" {
		err := conf.Load(*confFile)
		check(err)
	}
	r, err := repo.New()
	check(err)
	defer r.Close()
	u := &domain.User{Username: *user}
	u.SetPassword(*pass)
	err = r.SetUser(u)
	check(err)
}

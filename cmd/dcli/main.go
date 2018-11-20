package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/demisto/download/domain"
	"strconv"
)

var (
	user     = flag.String("u", "admin", "The user to create")
	pass     = flag.String("p", "", "The password to set")
	server   = flag.String("s", "https://download.demisto.com", "The location of the server")
	insecure = flag.Bool("insecure", false, "Skip cetificate check")
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
	if *user == "" {
		stderr("Please provide the username")
	}
	if *pass == "" {
		stderr("Please provide the password")
	}
	args := flag.Args()
	if len(args) == 0 {
		stderr("Please provide the action you want to perform")
	}
	c, err := New(*user, *pass, *server, *insecure)
	u, err := c.Login()
	check(err)
	fmt.Printf("Logged in with user %s [%s]\n", u.Username, u.Name)
	switch args[0] {
	case "tokens":
		tokens, err := c.Tokens()
		check(err)
		fmt.Println("Token\t\tDownloads")
		for _, t := range tokens {
			fmt.Printf("%s\t\t%d\n", t.Name, t.Downloads)
		}
	case "newu":
		if len(args) < 4 {
			stderr("User syntax is: 0 username password [name [email]] OR 1 token email [name]\n")
		}
		var u *userDetails
		if args[1] == "0" {
			u = &userDetails{Username: args[2], Password: args[3], Type: domain.UserTypeAdmin}
			if len(args) > 4 {
				u.Name = args[4]
			}
			if len(args) > 5 {
				u.Email = args[5]
			}
		} else {
			u = &userDetails{Username: args[2] + "*-*" + args[3], Token: args[2], Email: args[3]}
			if len(args) > 4 {
				u.Name = args[4]
			}
		}
		res, err := c.SetUser(u)
		check(err)
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("Created user:\n%s\n", string(b))
	case "email":
		if len(args) < 2 {
			stderr("Email syntax is: email [downloads] where downloads default is 3\n")
		}
		d := "3"
		if len(args) >= 3 {
			d = args[2]
		}
		downloads, err := strconv.Atoi(d)
		check(err)
		res, err := c.GenerateForEmail(args[1], downloads)
		check(err)
		fmt.Printf("Generated token %s with %d downloads\n", res.Name, res.Downloads)
		fmt.Printf("Link to download is https://download.demisto.com/download-params?token=%s&email=%s\n", res.Name, args[1])
	case "upload":
		if len(args) < 3 {
			stderr("Upload should receive 2 parameters - name and path\n")
		}
		err := c.Upload(args[1], args[2])
		check(err)
	case "gen":
		if len(args) < 3 {
			stderr("To generate tokens, please provide count and downloads\n")
		}
		count, err := strconv.Atoi(args[1])
		check(err)
		downloads, err := strconv.Atoi(args[2])
		check(err)
		tokens, err := c.Generate(count, downloads)
		check(err)
		fmt.Println("Token\t\tDownloads")
		for _, t := range tokens {
			fmt.Printf("%s\t\t%d\n", t.Name, t.Downloads)
		}
	case "log":
		l, err := c.DownloadLog()
		check(err)
		b, _ := json.MarshalIndent(l, "", "  ")
		fmt.Printf("%s\n", string(b))
	case "downloads":
		d, err := c.ListDownloads()
		check(err)
		for _, dn := range d {
			fmt.Printf("%20s%100s\t%v\n", dn.Name, dn.Path, dn.ModifyDate)
		}
	}
}

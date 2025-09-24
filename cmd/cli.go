package main

import (
	"flag"
	"fmt"
	"github.com/denverquane/slickshift/shift"
	"log"
)

func main() {
	var email string
	var password string
	flag.StringVar(&email, "email", "", "Email")
	flag.StringVar(&password, "password", "", "Password")
	flag.Parse()

	if email == "" {
		log.Fatal("-email is required")
	}

	if password == "" {
		log.Fatal("-password is required")
	}

	c, err := shift.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	cookie, err := c.Login(email, password)
	if err != nil {
		log.Fatal(err)
	}
	if cookie == "" {
		log.Fatal("empty cookie returned from login")
	} else {
		log.Println("Login successful!")
	}

	statusText, err := c.RedeemCode(cookie, "THRBT-WW6CB-56TB5-3B3BJ-XBW3X", "steam")
	if err != nil {
		log.Fatal(err)
	}
	status := shift.DetermineResponseType(statusText)
	fmt.Printf("Status Value: %d\nText: %s", status, statusText)
}

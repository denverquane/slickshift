package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/denverquane/slickshift/data"
	"github.com/denverquane/slickshift/shift"
)

// fatal logs the error and exits
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	var email string
	var password string
	flag.StringVar(&email, "email", "", "Email")
	flag.StringVar(&password, "password", "", "Password")
	flag.Parse()

	if email == "" {
		fatal("-email is required")
	}

	if password == "" {
		fatal("-password is required")
	}

	slog.Info("Default codes", "codes", data.DefaultBL4Codes())

	c, err := shift.NewClient(nil)
	if err != nil {
		fatal("Error creating client", "error", err.Error())
	}

	err = c.Login(email, password)
	if err != nil {
		fatal("Error logging in", "error", err.Error())
	}

	cookies := c.DumpCookies()
	if len(cookies) == 0 {
		fatal("No cookies returned")
	}

	// remake the client so we can verify that simply copying over the cookies will suffice
	c, err = shift.NewClient(cookies)
	if err != nil {
		fatal("Error creating client with cookies", "error", err.Error())
	}

	codes := data.DefaultBL4Codes()
	for code := range codes {
		status, err := c.RedeemCode(code, shift.Steam)
		if err != nil {
			slog.Error("Couldn't redeem code", "code", code, "error", err.Error())
		}
		statusCode := shift.DetermineResponseType(status)
		if statusCode == shift.Success {
			slog.Info("Redeemed code successfully", "code", code)
		} else {
			slog.Info("Redeeming code failed", "code", code, "status", status)
		}
	}

	//rewards, err := c.CheckRewards(shift.Steam, shift.Borderlands4)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//statusText, err := c.RedeemCode("JSX3J-B6SBJ-CXTBC-B3T3B-BZZZT", shift.Steam)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//status := shift.DetermineResponseType(statusText)
	//fmt.Printf("Status Value: %d\nText: %s", status, statusText)
}

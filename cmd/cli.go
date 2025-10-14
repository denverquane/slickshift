package main

import (
	"flag"
	"log"

	"github.com/denverquane/slickshift/data"
	"github.com/denverquane/slickshift/shift"
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

	log.Println(data.DefaultBL4Codes())

	c, err := shift.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	err = c.Login(email, password)
	if err != nil {
		log.Fatal(err)
	}

	cookies := c.DumpCookies()
	if len(cookies) == 0 {
		log.Fatal("No cookies returned")
	}

	// remake the client so we can verify that simply copying over the cookies will suffice
	c, err = shift.NewClient(cookies)
	if err != nil {
		log.Fatal(err)
	}

	codes := data.DefaultBL4Codes()
	for code := range codes {
		status, err := c.RedeemCode(code, shift.Steam)
		if err != nil {
			log.Println("Couldn't redeem code with error:", err)
		}
		statusCode := shift.DetermineResponseType(status)
		if statusCode == shift.Success {
			log.Printf("Redeemed %s successfully\n", code)
		} else {
			log.Printf("Redeemed %s failed with status: \"%s\"\n", code, status)
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

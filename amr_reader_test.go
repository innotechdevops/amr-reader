package amrreader_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/goccy/go-json"
	amrreader "github.com/innotechdevops/amr-reader"
	"github.com/prongbang/callx"
)

func TestAMR(t *testing.T) {
	callX := callx.New(callx.Config{
		Timeout:            60,
		Cookies:            true,
		InsecureSkipVerify: true,
	})

	config := amrreader.Config{
		BaseURL: os.Getenv("AMR_BASE_URL"),
		Logger:  true,
	}
	amr := amrreader.New(config, callX)

	// Read last cookies
	b, err := os.ReadFile(".cookies")
	if err != nil {
		t.Error("can't read the .cookies")
	}
	session := amrreader.Session{
		Username: os.Getenv("AMR_USERNAME"),
	}
	err = json.Unmarshal(b, &session.Header)
	if err != nil {
		t.Error("can't parse the .cookies to map")
	}

	// Get account
	acc, err := amr.GetAccount(session)
	if err != nil {
		session, err := amr.Auth(amrreader.Credential{
			Username: os.Getenv("AMR_USERNAME"),
			Password: os.Getenv("AMR_PASSWORD"),
		})

		fmt.Println("host:", config.Hostname())
		fmt.Println("session:", session)
		fmt.Println("error:", err)
		if err == nil {
			b, err := json.Marshal(session.Header)
			if err != nil {
				t.Error("can't convert to bytes")
			}
			err = os.WriteFile(".cookies", b, 0755)
			if err != nil {
				t.Error("can't write to file")
			}
		}
	}
	fmt.Println("account:", acc)
}

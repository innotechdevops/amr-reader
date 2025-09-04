package amrreader_test

import (
	"fmt"
	"os"
	"testing"

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
	account, err := amr.Auth(amrreader.Credential{
		Username: os.Getenv("AMR_USERNAME"),
		Password: os.Getenv("AMR_PASSWORD"),
	})

	fmt.Println("host:", config.Hostname())
	fmt.Println("account:", account)
	fmt.Println("error:", err)
}

package amrreader_test

import (
	"fmt"
	"testing"

	amrreader "github.com/innotechdevops/amr-reader"
	"github.com/prongbang/callx"
)

func TestAMR(t *testing.T) {
	callX := callx.New(callx.Config{Timeout: 60})

	config := amrreader.Config{
		BaseURL: "https://www.example.com",
		Logger:  true,
	}
	amr := amrreader.New(config, callX)
	account, err := amr.Auth(amrreader.Credential{
		Username: "u",
		Password: "p",
	})

	fmt.Println("host:", config.Hostname())
	fmt.Println("account:", account)
	fmt.Println("error:", err)
}

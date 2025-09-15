package amrreader_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/goccy/go-json"
	amrreader "github.com/innotechdevops/amr-reader"
	"github.com/prongbang/callx"
)

func TestAuth(t *testing.T) {
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

	session, err := amr.Auth(amrreader.Credential{
		Username: os.Getenv("AMR_USERNAME"),
		Password: os.Getenv("AMR_PASSWORD"),
	})
	if err != nil {
		t.Error(err)
	}

	// Write header to file
	b, err := json.Marshal(session.Header)
	if err != nil {
		t.Error("can't convert to bytes")
	}
	err = os.WriteFile(".header", b, 0755)
	if err != nil {
		t.Error("can't write to file")
	}
}

func TestGetAccount(t *testing.T) {
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

	// Read last header
	b, err := os.ReadFile(".header")
	if err != nil {
		t.Error("can't read the .header")
	}
	session := amrreader.Session{
		Username: os.Getenv("AMR_USERNAME"),
	}
	err = json.Unmarshal(b, &session.Header)
	if err != nil {
		t.Error("can't parse the .header to map")
	}

	// Get account
	acc, err := amr.GetAccount(session)
	fmt.Println("error:", err)
	fmt.Println("account:", acc)
}

func TestPasswordRotation(t *testing.T) {
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

	// Read last header
	b, err := os.ReadFile(".header")
	if err != nil {
		t.Error("can't read the .header")
	}
	session := amrreader.Session{
		Username: os.Getenv("AMR_USERNAME"),
	}
	err = json.Unmarshal(b, &session.Header)
	if err != nil {
		t.Error("can't parse the .header to map")
	}

	// Change password
	credential := amrreader.PasswordRotationCredential{
		Username:    session.Username,
		OldPassword: os.Getenv("AMR_PASSWORD"),
		NewPassword: os.Getenv("AMR_NEW_PASSWORD"),
	}
	newSession, err := amr.PasswordRotation(credential, &session.Header)
	fmt.Println("err:", err)
	fmt.Println("newSession:", newSession)
	if err == nil {
		fmt.Println("password changed:", credential.NewPassword)
	}
}

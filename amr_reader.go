package amrreader

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/innotechdevops/core"
	"github.com/innotechdevops/timex"
	"github.com/pkg/errors"
	"github.com/prongbang/callx"
)

var hiddenIdList = []string{"__EVENTTARGET", "__EVENTARGUMENT", "__LASTFOCUS", "__VIEWSTATE", "__VIEWSTATEGENERATOR", "__EVENTVALIDATION"}

type Config struct {
	BaseURL string
	Logger  bool
}

func (c Config) Hostname() string {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return c.BaseURL
	}
	return u.Hostname()
}

type Credential struct {
	Username string
	Password string
}

type PasswordRotationCredential struct {
	Username    string
	OldPassword string
	NewPassword string
}

func (c *Credential) Checksum() string {
	sh := sha1.New()
	sh.Write([]byte(c.Username + ":" + c.Password))
	return fmt.Sprintf("%x", sh.Sum(nil))
}

type Account struct {
	Header       callx.Header
	MeterNo      string
	MeterPoint   string
	CustomerId   string
	CustomerCode string
}

type Profile struct {
	Time              *time.Time `json:"time"`
	EnergyConsumption *float64   `json:"energyConsumption"`
}

type ProfileMeter struct {
	CustomerId   string    `json:"customerId"`
	CustomerCode string    `json:"customerCode"`
	MeterNo      string    `json:"meterNo"`
	MeterPoint   string    `json:"meterPoint"`
	Profile      []Profile `json:"profile"`
}

type Session struct {
	Header   callx.Header
	Username string
}

type AmrX interface {
	Auth(config Credential) (*Session, error)
	GetAccount(session Session) (*Account, error)
	GetProfileDaily(acc Account, date string) (*ProfileMeter, error)
	PasswordRotation(config PasswordRotationCredential, header *map[string]string) (*Session, error)
}

type amrX struct {
	Config  Config
	CallX   callx.CallX
	session callx.Header
	logger  Logger
}

func Header(hostname string) callx.Header {
	return callx.Header{
		"sec-ch-ua":                 "\"Chromium\";v=\"128\", \"Not;A=Brand\";v=\"24\", \"Google Chrome\";v=\"128\"",
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        "\"macOS\"",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		"Sec-Fetch-Site":            "same-origin",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
		"host":                      hostname,
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
	}
}

func (a *amrX) PasswordRotation(config PasswordRotationCredential, header *map[string]string) (*Session, error) {
	var session *Session
	var account *Account
	if header != nil {
		acc, err := a.GetAccount(Session{
			Header:   *header,
			Username: config.Username,
		})
		if err != nil {
			sess, err := a.Auth(Credential{
				Username: config.Username,
				Password: config.OldPassword,
			})
			if err != nil {
				return nil, err
			}
			session = sess
			acc, err = a.GetAccount(*sess)
			if err != nil {
				return nil, err
			}
			account = acc
		} else {
			account = acc
		}
	}

	fmt.Println(account)

	// Pre-Change Password
	query := map[string]string{
		"Custid":   account.CustomerId,
		"CustCode": account.CustomerCode,
	}
	custom := callx.Custom{
		URL:    a.Config.BaseURL + "/ChgPassword.aspx?" + core.ToQuery(query),
		Method: http.MethodGet,
		Header: account.Header,
	}

	a.logger.Info("Pre-Change Password:")
	a.logger.Info("> Request:")
	a.logger.Info(fmt.Sprintf("	POST -> %s", custom.URL))

	preChangePassRs := a.CallX.Req(custom)

	a.logger.Info("> Response:")
	a.logger.Info(fmt.Sprintf("	Cookie %s", preChangePassRs.Cookies))

	// Parse html string to dom
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(preChangePassRs.Data)))
	if err != nil {
		a.logger.Fatal("Error loading HTML:", err)
	}

	// Post-Change Password
	form := url.Values{
		"txtPassword": {config.OldPassword},
		"txtNewPass":  {config.NewPassword},
		"txtConfPass": {config.NewPassword},
		"btnSubmit":   {"ตกลง"},
	}

	// Hidden input
	hdnId, hdnIdExist := doc.Find("#hdnId").Attr("value")
	if hdnIdExist {
		form.Set("hdnId", hdnId)
	}
	for _, id := range hiddenIdList {
		value, exists := doc.Find(fmt.Sprintf("#%s", id)).Attr("value")
		if exists {
			form.Set(id, value)
		}
	}
	custom.Header["Content-Type"] = "application/x-www-form-urlencoded"

	custom.Method = http.MethodPost
	custom.Form = strings.NewReader(form.Encode())
	data := a.CallX.Req(custom)
	if data.Code == http.StatusOK {
		return session, nil
	}

	return session, errors.New("cannot change password")
}

// GetProfileDaily
// date is supported format "19/09/2024"
func (a *amrX) GetProfileDaily(acc Account, date string) (*ProfileMeter, error) {
	loc, _ := time.LoadLocation(timex.TimeZoneAsiaBangkok)

	query := map[string]string{
		"Custid":     acc.CustomerId,
		"CustCode":   acc.CustomerCode,
		"MeterPoint": acc.MeterPoint,
		"RepDate":    date,
		"SumMeter":   "0",
		"GrphType":   "Col",
		"DataType":   "0",
		"kWh":        "1", // กิโลวัตต์ชั่วโมง (kWh)
		"kVarh":      "0", // กิโลวาร์ชั่วโมง (kVarh)
		"kW":         "0", // กิโลวัตต์ (kW)
		"kVar":       "0", // กิโลวาร์ (kVar)
		"kWh1":       "0",
		"kVarh1":     "0",
		"kW1":        "0",
		"kVar1":      "0",
		"Cur":        "0",
		"Vol":        "0",
		"PF":         "0",
		"PD":         "0",
		"chk":        "0",
	}

	custom := callx.Custom{
		URL:    a.Config.BaseURL + "/showDailyProfile.aspx?" + core.ToQuery(query),
		Method: http.MethodGet,
		Header: acc.Header,
	}
	data := a.CallX.Req(custom)

	// Parse html string to dom
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data.Data)))
	if err != nil {
		a.logger.Info("Error loading HTML:", "error", err.Error())
		return nil, err
	}

	divTable := doc.Find("#divTable")

	profileMeter := ProfileMeter{
		CustomerId:   acc.CustomerId,
		CustomerCode: acc.CustomerCode,
		MeterNo:      acc.MeterNo,
		MeterPoint:   acc.MeterPoint,
		Profile:      []Profile{},
	}

	// Find the <tr> and iterate over its <td> elements
	maxCol := 0
	divTable.Find("tr").Each(func(row int, s *goquery.Selection) {
		profile := Profile{}

		s.Find("td").Each(func(col int, td *goquery.Selection) {
			if row == 0 {
				maxCol = col
			} else {
				text := td.Text()
				if col == 0 {
					parsedTime, t := time.ParseInLocation("02/01/2006 15.04", text, loc)
					if t == nil {
						profile.Time = &parsedTime
					}
				} else if col == maxCol {
					floatVal, fErr := strconv.ParseFloat(strings.ReplaceAll(text, ",", ""), 64)
					if fErr == nil {
						profile.EnergyConsumption = &floatVal
					}
				}
			}
		})

		if profile.Time != nil {
			profileMeter.Profile = append(profileMeter.Profile, profile)
		}
	})

	a.logger.Info("GetProfileDaily", "date", date)

	return &profileMeter, nil
}

func (a *amrX) Auth(config Credential) (*Session, error) {
	// Pre-Auth
	custom := callx.Custom{
		URL:    a.Config.BaseURL + "/Index.aspx",
		Method: http.MethodGet,
		Header: Header(a.Config.Hostname()),
	}

	a.logger.Info("Pre-Auth:")
	a.logger.Info("> Request:")
	a.logger.Info(fmt.Sprintf("	POST -> %s", custom.URL))

	preAuthRs := a.CallX.Req(custom)

	a.logger.Info("> Response:")
	a.logger.Info(fmt.Sprintf("	Cookie %s", preAuthRs.Cookies))

	// Parse html string to dom
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(preAuthRs.Data)))
	if err != nil {
		a.logger.Fatal("Error loading HTML:", err)
	}

	// Post-Auth
	form := url.Values{
		"btnOK":       {"เข้าสู่ระบบ"},
		"txtUsername": {config.Username},
		"txtPassword": {config.Password},
		"ddlLanguage": {"th-TH"},
	}

	a.logger.Info("Auth:")
	a.logger.Info("> Request:")
	a.logger.Info(fmt.Sprintf("	Header: %s", custom.Header))
	a.logger.Info(fmt.Sprintf("	Form: %s", form.Encode()))

	// Hidden input
	for _, id := range hiddenIdList {
		value, exists := doc.Find(fmt.Sprintf("#%s", id)).Attr("value")
		if exists {
			form.Set(id, value)
		}
	}
	custom.Header["Content-Type"] = "application/x-www-form-urlencoded"

	// Get cookies session id
	re := regexp.MustCompile(`ASP\.NET_SessionId=([^;]+)`)
	match := re.FindStringSubmatch(preAuthRs.Cookies["ASP.NET_SessionId"])
	if len(match) > 1 {
		custom.Header["Cookie"] = fmt.Sprintf("ASP.NET_SessionId=%s", match[1])
	}
	custom.Method = http.MethodPost
	custom.Form = strings.NewReader(form.Encode())

	a.logger.Info("Post-Auth:")
	a.logger.Info("> Request:")
	a.logger.Info(fmt.Sprintf("	POST -> %s", custom.URL))
	a.logger.Info(fmt.Sprintf("	Header: %s", custom.Header))
	a.logger.Info(fmt.Sprintf("	Form: %s", form.Encode()))

	postAuthRs := a.CallX.Req(custom)

	a.logger.Info("> Response:")
	a.logger.Info(fmt.Sprintf("	Status %d %s", postAuthRs.Code, http.StatusText(postAuthRs.Code)))
	a.logger.Info(fmt.Sprintf("	Cookie %s", custom.Header["Cookie"]))

	if postAuthRs.Code == http.StatusFound {
		return &Session{Header: custom.Header, Username: config.Username}, nil
	} else {
		a.logger.Error("Auth failure")
	}

	// reset session
	a.session = callx.Header{}

	return nil, errors.New("ไม่สามารถเข้าสู่ระบบได้: รหัสผ่านใกล้หมดอายุหรือหมดอายุแล้ว")
}

func (a *amrX) GetAccount(session Session) (*Account, error) {
	mainCustCustom := callx.Custom{URL: a.Config.BaseURL + "/MainCust.aspx", Method: http.MethodGet, Header: session.Header}

	a.logger.Info("Main Customer:")
	a.logger.Info("> Request:")
	a.logger.Info(fmt.Sprintf("	GET -> %s", mainCustCustom.URL))
	a.logger.Info(fmt.Sprintf("	Header: %s", mainCustCustom.Header))

	mainCustRs := a.CallX.Req(mainCustCustom)
	mainCustHtml := string(mainCustRs.Data)

	a.logger.Info("> Response:")
	a.logger.Info(fmt.Sprintf("	Status %d %s", mainCustRs.Code, http.StatusText(mainCustRs.Code)))

	// Parse html string to dom
	docCust, errCust := goquery.NewDocumentFromReader(strings.NewReader(mainCustHtml))
	if errCust != nil {
		a.logger.Fatal("Error loading HTML:", errCust)
	}

	/// Get url from iframe: <iframe id="frmMain" name="frmMain" src="..."
	custUrl, exists := docCust.Find("#frmMain").Attr("src")
	if exists {
		custUrl = strings.ReplaceAll(custUrl, "./", a.Config.BaseURL+"/")
		queryParams := core.Query(custUrl)
		custID := queryParams.Get("Custid")
		meterNo := queryParams.Get("PeaNo")
		custDashboardCustom := callx.Custom{URL: custUrl, Method: http.MethodGet, Header: session.Header}

		a.logger.Info("Customer Dashboard Custom:")
		a.logger.Info("> Request:")
		a.logger.Info(fmt.Sprintf("	GET -> %s", custDashboardCustom.URL))
		a.logger.Info(fmt.Sprintf("	Header: %s", custDashboardCustom.Header))

		// Get meter no and meter point
		custDashboardRs := a.CallX.Req(custDashboardCustom)

		a.logger.Info("> Response:")
		a.logger.Info(fmt.Sprintf("	Status %d %s", mainCustRs.Code, http.StatusText(mainCustRs.Code)))

		// Parse html string to dom
		docDash, errDash := goquery.NewDocumentFromReader(strings.NewReader(string(custDashboardRs.Data)))
		if errDash != nil {
			a.logger.Fatal("Error loading HTML:", errCust)
		}

		meterPoint := docDash.Find("select#ddlMeter option[selected]").AttrOr("value", "")
		meterNo = docDash.Find("select#ddlMeter option[selected]").Text()

		// reset session
		a.session = session.Header

		return &Account{
			Header:       session.Header,
			CustomerId:   custID,
			CustomerCode: session.Username,
			MeterNo:      meterNo,
			MeterPoint:   meterPoint,
		}, nil
	} else {
		a.logger.Error("Id frmMain not found")
	}
	return nil, errors.New("cannot get customer information")
}

func New(config Config, callX callx.CallX) AmrX {
	return &amrX{
		Config:  config,
		CallX:   callX,
		session: callx.Header{},
		logger:  Logger{Enabled: config.Logger},
	}
}

package extra

import (
	"net/url"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
)

func init() {
	seabird.RegisterPlugin("fcc", newFccPlugin)
}

type fccPlugin struct{}

type fccLicense struct {
	Name       string `json:"licName"`
	Frn        string `json:"frn"`
	Callsign   string `json:"callsign"`
	Category   string `json:"categoryDesc"`
	Service    string `json:"serviceDesc"`
	Status     string `json:"statusDesc"`
	ExpireDate string `json:"expiredDate"`
	LicenseID  string `json:"licenseID"`
	LicenseURL string `json:"licDetailURL"`
}

type fccLicenses struct {
	Page       string       `json:"page"`
	RowPerPage string       `json:"rowPerPage"`
	TotalRows  string       `json:"totalRows"`
	LastUpdate string       `json:"lastUpdate"`
	Licenses   []fccLicense `json:"License"`
}

type fccResponse struct {
	Status      string      `json:"status"`
	LicenseData fccLicenses `json:"Licenses"`
}

func newFccPlugin(b *seabird.Bot) error {
	cm := b.CommandMux()

	p := &fccPlugin{}

	cm.Event("callsign", p.Search, &seabird.HelpInfo{
		Usage:       "<callsign>",
		Description: "Finds information about given FCC callsign",
	})

	return nil
}

func (p *fccPlugin) Search(r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			r.MentionReplyf("Callsign required")
			return
		}

		url := "http://data.fcc.gov/api/license-view/basicSearch/getLicenses?format=json&searchValue=" + url.QueryEscape(r.Message.Trailing())

		fr := &fccResponse{}
		err := internal.GetJSON(url, fr)
		if err != nil {
			r.MentionReplyf("%s", err)
			return
		}

		if len(fr.LicenseData.Licenses) == 0 {
			r.MentionReplyf("No licenses found")
			return
		}

		license := fr.LicenseData.Licenses[0]
		r.MentionReplyf("%s (%s): %s, %s, expires %s", license.Callsign, license.Service, license.Name, license.Status, license.ExpireDate)
	}()
}

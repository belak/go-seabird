package extra

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("watchdog", newWatchdogPlugin)
}

type watchdogPlugin struct {
	db *xorm.Engine
}

type watchdogCheck struct {
	Time   time.Time `xorm:"created"`
	Entity string
	Nonce  string
}

type CheckRequest struct {
	Nonce string `json:"nonce"`
	UseDb bool   `json:"use_db,omitempty"`
}

type CheckResponse struct {
	Time    int64  `json:"time"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Nonce   string `json:"nonce,omitempty"`
}

func newWatchdogPlugin(b *seabird.Bot, m *seabird.BasicMux, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &watchdogPlugin{db: db}

	// Migrate any relevant tables
	err := db.Sync(watchdogCheck{})
	if err != nil {
		return err
	}

	cm.Event("watchdog-check", p.check, &seabird.HelpInfo{
		Description: "Used to check availability of Seabird optionally including its DB",
	})

	return nil
}

func (p *watchdogPlugin) marshalOrRespondFailedMarshal(b *seabird.Bot, r *seabird.Request, response *CheckResponse) {
	response.Time = time.Now().Unix()
	respStr, err := json.Marshal(response)

	if err != nil {
		b.MentionReply(r, `{"time":%d,"status":"failed to marshal response"}`, response.Time)
		return
	}

	b.MentionReply(r, "%s", respStr)
}

func (p *watchdogPlugin) checkDb(b *seabird.Bot, r *seabird.Request, request *CheckRequest) bool {
	if !request.UseDb {
		return true
	}

	check := &watchdogCheck{
		Entity: r.Message.Prefix.String(),
		Nonce:  request.Nonce,
	}

	_, err := p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		return s.Insert(check)
	})

	if err != nil {
		resp := &CheckResponse{
			Success: false,
			Message: fmt.Sprintf("Error writing check to DB: \"%s\"", err),
			Nonce:   request.Nonce,
		}

		p.marshalOrRespondFailedMarshal(b, r, resp)

		return false
	}

	return true
}

func (p *watchdogPlugin) check(b *seabird.Bot, r *seabird.Request) {
	timer := r.Timer("watchdog-check")
	defer timer.Done()

	if len(r.Message.Trailing()) == 0 {
		resp := &CheckResponse{
			Success: false,
			Message: "missing json argument",
		}
		p.marshalOrRespondFailedMarshal(b, r, resp)

		return
	}

	request := CheckRequest{
		UseDb: true,
	}

	err := json.Unmarshal([]byte(r.Message.Trailing()), &request)
	if err != nil {
		resp := &CheckResponse{
			Success: false,
			Message: fmt.Sprintf("unable to parse JSON: \"%s\"", err),
		}
		p.marshalOrRespondFailedMarshal(b, r, resp)

		return
	}

	if len(request.Nonce) == 0 {
		resp := &CheckResponse{
			Success: false,
			Message: fmt.Sprintf("missing nonce"),
		}
		p.marshalOrRespondFailedMarshal(b, r, resp)

		return
	}

	ok := p.checkDb(b, r, &request)
	if !ok {
		return
	}

	resp := &CheckResponse{
		Success: true,
		Message: "ok",
		Nonce:   request.Nonce,
	}

	p.marshalOrRespondFailedMarshal(b, r, resp)
}

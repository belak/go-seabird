package extra

import (
	"strconv"
	"time"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("weight_tracker", newWeightPlugin)
}

type weightPlugin struct {
	db *xorm.Engine
}

type Measurement struct {
	Name   string
	Date   time.Time `xorm:"created"`
	Weight float64
}

func newWeightPlugin(b *seabird.Bot, m *seabird.BasicMux, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &weightPlugin{db: db}

	// Migrate any relevant tables
	err := db.Sync(Measurement{})
	if err != nil {
		return err
	}

	cm.Event("add-weight", p.addWeight, &seabird.HelpInfo{
		Usage:       "<value>",
		Description: "Adds a new weight measurement for the current user",
	})

	cm.Event("last-weight", p.lastWeight, &seabird.HelpInfo{
		Usage:       "",
		Description: "Gets the most recent weight measurement for the current user",
	})

	return nil
}

func (p *weightPlugin) addWeight(b *seabird.Bot, r *seabird.Request) {
	if len(r.Message.Trailing()) == 0 {
		b.MentionReply(r, "You must specify a new weight measurement")
		return
	}

	weight, err := strconv.ParseFloat(r.Message.Trailing(), 64)
	if err != nil {
		b.MentionReply(r, "Invalid weight measurement")
		return
	}

	measurement := &Measurement{Name: r.Message.Prefix.Name, Weight: weight}

	p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		res, err := s.Insert(measurement)
		if err != nil {
			b.MentionReply(r, "Error inserting new weight measurement: %v", err)
		}
		return res, err
	})

	b.MentionReply(r, "Measurement added")
}

func (p *weightPlugin) lastWeight(b *seabird.Bot, r *seabird.Request) {
	user := r.Message.Prefix.Name
	measurement := &Measurement{Name: user}

	_, err := p.db.Desc("date").Limit(1).Get(measurement)
	if err != nil {
		b.MentionReply(r, "Error fetching measurement value: %v", err)
		return
	}

	b.MentionReply(r, "Last measurement for %s was %.2f lbs", user, measurement.Weight)
}
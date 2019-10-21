package extra

import (
	"strconv"
	"time"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
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

	/*
	   cm.Event("weight-delta", p.weightDelta, &seabird.HelpInfo{
	           Usage:       "[start_date]",
	           Description: "Gets the current delta from either the first submitted measurement or from the submitted start date",
	   })

	   cm.Event("weight-url", p.weightUrl, &seabird.HelpInfo{
	           Usage:       "",
	           Description: "Gets the URL for viewing the current user's weight graph",
	   })
	*/

	return nil
}

func (p *weightPlugin) addWeight(b *seabird.Bot, m *irc.Message) {
	if len(m.Trailing()) == 0 {
		b.MentionReply(m, "You must specify a new weight measurement")
		return
	}

	weight, err := strconv.ParseFloat(m.Trailing(), 64)
	if err != nil {
		b.MentionReply(m, "Invalid weight measurement")
		return
	}

	measurement := &Measurement{Name: m.Prefix.Name, Weight: weight}

	p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		res, err := s.Insert(measurement)
		if err != nil {
			b.MentionReply(m, "Error inserting new weight measurement: %v", err)
		}
		return res, err
	})

	b.MentionReply(m, "Measurement added")
}

func (p *weightPlugin) lastWeight(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name
	measurement := &Measurement{Name: user}

	_, err := p.db.Desc("date").Limit(1).Get(measurement)
	if err != nil {
		b.MentionReply(m, "Error fetching measurement value: %v", err)
		return
	}

	b.MentionReply(m, "Last measurement for %s was %.2f lbs", user, measurement.Weight)
}

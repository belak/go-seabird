package extra

import (
	"context"
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

func newWeightPlugin(b *seabird.Bot) error {
	p := &weightPlugin{
		db: CtxDB(b.Context()), // TODO: ensure DB loaded
	}

	// Migrate any relevant tables
	err := p.db.Sync(Measurement{})
	if err != nil {
		return err
	}

	cm := b.CommandMux()

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

func (p *weightPlugin) addWeight(ctx context.Context, r *seabird.Request) {
	if len(r.Message.Trailing()) == 0 {
		r.MentionReply("You must specify a new weight measurement")
		return
	}

	weight, err := strconv.ParseFloat(r.Message.Trailing(), 64)
	if err != nil {
		r.MentionReply("Invalid weight measurement")
		return
	}

	measurement := &Measurement{Name: r.Message.Prefix.Name, Weight: weight}

	p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		res, err := s.Insert(measurement)
		if err != nil {
			r.MentionReply("Error inserting new weight measurement: %v", err)
		}
		return res, err
	})

	r.MentionReply("Measurement added")
}

func (p *weightPlugin) lastWeight(ctx context.Context, r *seabird.Request) {
	user := r.Message.Prefix.Name
	measurement := &Measurement{Name: user}

	_, err := p.db.Desc("date").Limit(1).Get(measurement)
	if err != nil {
		r.MentionReply("Error fetching measurement value: %v", err)
		return
	}

	r.MentionReply("Last measurement for %s was %.2f lbs", user, measurement.Weight)
}

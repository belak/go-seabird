package extra

import (
	"math/rand"
	"strings"
	"time"

	"github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/extra/uno"
	"github.com/belak/irc"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("uno", newUnoPlugin)

	rand.Seed(time.Now().UTC().UnixNano())
}

type unoPlugin struct {
	db   *nut.DB
	game *uno.UnoGame
}

// Buckets:
// - players
// - decks
func newUnoPlugin(cm *seabird.CommandMux, db *nut.DB) error {
	p := &unoPlugin{db: db}

	err := p.db.EnsureBucket("uno")
	if err != nil {
		return err
	}

	cm.Event("uno", p.unoCallback, &seabird.HelpInfo{
		Usage:       "tbd",
		Description: "UNO command",
	})

	cm.Event("hand", p.getHandCallback, &seabird.HelpInfo{
		Usage:       "hand",
		Description: "Messages you your hand in an UNO game",
	})

	return nil
}

func (p *unoPlugin) unoCallback(b *seabird.Bot, m *irc.Message) {
	trailing := m.Trailing()
	if trailing == "" {
		b.MentionReply(m, "Usage: <prefix>uno start")
		return
	}

	args := strings.Split(trailing, " ")
	switch args[0] {
	case "start":
		p.startCallback(b, m, args[1:])
	default:
		b.MentionReply(m, "Unknown command \"%s\"", args[0])
		return
	}
}

func (p *unoPlugin) firstTurn(b *seabird.Bot, m *irc.Message) {
	for _, message := range p.game.FirstTurn() {
		b.MentionReply(m, message)
	}
}

func (p *unoPlugin) startCallback(b *seabird.Bot, m *irc.Message, args []string) {
	if p.game != nil {
		b.MentionReply(m, "Game already running")
		return
	}
	if len(args) < 2 {
		b.MentionReply(m, "Must provide at least two players")
		return
	}

	logger := b.GetLogger()
	game, err := uno.NewGame(args)
	p.game = game
	if err != nil {
		b.MentionReply(m, "Unable to start UNO game: ", err)
	}

	p.firstTurn(b, m)

	b.MentionReply(m, p.game.Discard.Top().String())
	for _, player := range p.game.Players {
		logger.Info(player.Name)
	}
}

func (p *unoPlugin) getHandCallback(b *seabird.Bot, m *irc.Message) {
	if p.game == nil {
		b.MentionReply(m, "No UNO game running.")
		return
	}

	player, err := p.game.GetPlayer(m.Prefix.Name)
	if err != nil {
		b.MentionReply(m, "Looks like you're not playing UNO right now. Try again next time!")
		return
	}

	if m.FromChannel() {
		b.MentionReply(m, "You probably don't want to show your hand to other players.")
		return
	}

	for _, card := range player.Hand.Cards {
		b.Reply(m, card.String())
	}
}

/*
func (p *unoPlugin) getKey(key string) (*phrase, error) {
	row := &phraseBucket{Key: p.cleanedName(key)}
	if len(row.Key) == 0 {
		return nil, errors.New("No key provided")
	}

	err := p.db.View(func(tx *nut.Tx) error {
		bucket := tx.Bucket("uno")
		return bucket.Get(row.Key, row)
	})

	if err != nil {
		return nil, err
	} else if len(row.Entries) == 0 {
		return nil, errors.New("No results for given key")
	}

	entry := row.Entries[len(row.Entries)-1]
	if entry.Deleted {
		return nil, errors.New("Phrase was previously deleted")
	}

	return &entry, nil
}
*/

/*
func (p *unoPlugin) forgetCallback(b *seabird.Bot, m *irc.Message) {
	row := &phraseBucket{Key: p.cleanedName(m.Trailing())}
	if len(row.Key) == 0 {
		b.MentionReply(m, "No key supplied")
	}

	entry := phrase{
		Submitter: m.Prefix.Name,
		Deleted:   true,
	}

	err := p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("uno")
		err := bucket.Get(row.Key, row)
		if err != nil {
			return errors.New("No results for given key")
		}

		row.Entries = append(row.Entries, entry)

		return bucket.Put(row.Key, row)
	})

	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "Forgot %s", row.Key)
}
*/

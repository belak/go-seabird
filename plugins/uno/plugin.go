package uno

import (
	"errors"
	"strings"

	"github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins"
	"github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("uno", newUnoPlugin)
}

type unoPlugin struct {
	games   map[string]*Game
	tracker *plugins.ChannelTracker
}

func newUnoPlugin(cm *seabird.CommandMux, tracker *plugins.ChannelTracker) {
	p := &unoPlugin{
		games:   make(map[string]*Game),
		tracker: tracker,
	}

	// TODO: Track channel parts

	cm.Channel("uno", p.unoCallback, &seabird.HelpInfo{
		Usage:       "[create|join|start|stop]",
		Description: "Flow control and stuff",
	})

	cm.Channel("hand", p.handCallback, &seabird.HelpInfo{
		Usage:       "hand",
		Description: "Messages you your hand in an UNO game",
	})

	cm.Channel("play", p.playCallback, &seabird.HelpInfo{
		Usage:       "play <hand_index>",
		Description: "Plays card from your hand at <hand_index> and ends your turn",
	})

	cm.Channel("draw", p.drawCallback, &seabird.HelpInfo{
		Usage:       "draw",
		Description: "Draws a card and possibly ends your turn",
	})

	cm.Channel("draw_play", p.drawPlayCallback, &seabird.HelpInfo{
		Usage:       "draw_play [yes|no]",
		Description: "Used after a call to <prefix>draw to possibly play a card",
	})

	cm.Channel("color", p.colorCallback, &seabird.HelpInfo{
		Usage:       "color red|yellow|green|blue",
		Description: "Selects next color to play",
	})
}

func (p *unoPlugin) lookupDataRaw(b *seabird.Bot, m *irc.Message) (*plugins.User, *Game) {
	user := p.tracker.LookupUser(m.Prefix.Name)
	game := p.games[m.Params[0]]

	return user, game
}

func (p *unoPlugin) lookupData(b *seabird.Bot, m *irc.Message) (*plugins.User, *Game, error) {
	user, game := p.lookupDataRaw(b, m)

	if user == nil {
		return user, game, errors.New("Couldn't find user")
	}

	if game == nil {
		return user, game, errors.New("No game in this channel")
	}

	return user, game, nil
}

// sendMessages is an abstraction around sending the uno Message
// type. This simplifies the translation between that and IRC.
func (p *unoPlugin) sendMessages(b *seabird.Bot, m *irc.Message, uMsgs []*Message) {
	for _, uMsg := range uMsgs {
		if uMsg.Target == nil {
			b.Reply(m, "%s", uMsg.Message)
		} else if uMsg.Private {
			b.Send(&irc.Message{
				Command: "PRIVMSG",
				Params: []string{
					uMsg.Target.Nick,
					uMsg.Message,
				},
			})
		} else {
			b.Reply(m, "%s: %s", uMsg.Target.Nick, uMsg.Message)
		}
	}
}

func (p *unoPlugin) unoCallback(b *seabird.Bot, m *irc.Message) {
	trailing := m.Trailing()
	if trailing == "" {
		b.MentionReply(m, "Usage: <prefix>uno [create|join|start|stop]")
		return
	}

	args := strings.Split(trailing, " ")

	if len(args) == 1 {
		p.rawUnoCallback(b, m)
		return
	}

	switch args[0] {
	case "create":
		p.createCallback(b, m)
	case "join":
		p.joinCallback(b, m)
	case "start":
		p.startCallback(b, m)
	case "stop":
		p.stopCallback(b, m)
	default:
		b.MentionReply(m, "Unknown command \"%s\"", args[0])
		return
	}
}

func (p *unoPlugin) rawUnoCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.SayUno(user))
}

func (p *unoPlugin) createCallback(b *seabird.Bot, m *irc.Message) {
	user, game := p.lookupDataRaw(b, m)
	if user == nil {
		b.MentionReply(m, "Couldn't find user")
		return
	}

	if game != nil {
		b.MentionReply(m, "There's already a game in this channel")
		return
	}

	// Create a new game, add the current user and store it.
	game, messages := NewGame(user)
	p.sendMessages(b, m, messages)
	p.games[m.Params[0]] = game
}

func (p *unoPlugin) joinCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.AddPlayer(user))
}

func (p *unoPlugin) startCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.Start(user))
}

func (p *unoPlugin) stopCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	messages, ok := game.Stop(user)

	p.sendMessages(b, m, messages)

	if ok {
		delete(p.games, m.Params[0])
	}
}

func (p *unoPlugin) handCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.GetHand(user))
}

func (p *unoPlugin) playCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	messages, done := game.Play(user, m.Trailing())
	if done {
		delete(p.games, m.Params[0])
	}

	p.sendMessages(b, m, messages)
}

func (p *unoPlugin) drawCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.Draw(user))
}

func (p *unoPlugin) drawPlayCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.DrawPlay(user, m.Trailing()))
}

func (p *unoPlugin) colorCallback(b *seabird.Bot, m *irc.Message) {
	user, game, err := p.lookupData(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	p.sendMessages(b, m, game.SetColor(user, m.Trailing()))
}

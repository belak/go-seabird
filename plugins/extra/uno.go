package extra

import (
	"fmt"
	"math/rand"
	"strconv"
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

func PrivateMessage(b *seabird.Bot, target, format string, v ...interface{}) {
	b.Send(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "PRIVMSG",
		Params: []string{
			target,
			fmt.Sprintf(format, v...),
		},
	})
}

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

	cm.Event("play", p.playCallback, &seabird.HelpInfo{
		Usage:       "play <hand_index>",
		Description: "Plays card from your hand at <hand_index> and ends your turn",
	})

	cm.Event("draw", p.drawCallback, &seabird.HelpInfo{
		Usage:       "draw",
		Description: "Draws a card and possibly ends your turn",
	})

	cm.Event("color", p.colorCallback, &seabird.HelpInfo{
		Usage:       "color red|yellow|green|blue",
		Description: "Selects next color to play",
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

func (p *unoPlugin) sendMessages(b *seabird.Bot, m *irc.Message, lines []string) {
	for _, line := range lines {
		b.Reply(m, line)
	}
}

func (p *unoPlugin) messageHand(b *seabird.Bot, m *irc.Message, player *uno.UnoPlayer) {
	PrivateMessage(b, player.Name, "<hand.summary>")
	for _, card := range player.Hand.Cards {
		PrivateMessage(b, player.Name, card.String())
	}
	PrivateMessage(b, player.Name, "</hand.summary>")
}

func (p *unoPlugin) messageHands(b *seabird.Bot, m *irc.Message) {
	for _, player := range p.game.Players {
		p.messageHand(b, m, player)
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
		return
	}

	p.messageHands(b, m)

	p.sendMessages(b, m, p.game.FirstTurn())

	for _, player := range p.game.Players {
		logger.Info(player.Name)
	}
}

func (p *unoPlugin) checkGame(b *seabird.Bot, m *irc.Message, desiredStates []uno.GameState) (string, bool) {
	if p.game == nil {
		return "No UNO game running.", false
	}
	if p.game.CurrentPlayer().Name != m.Prefix.Name {
		return "It's not your turn!", false
	}

	for _, state := range desiredStates {
		if p.game.State() == state {
			return "", true
		}
	}
	return "Oak's words echoed... There's a time and place for everything, but not now.", false
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

func ColorForString(colorStr string) uno.ColorCode {
	switch colorStr {
	case "red":
		return uno.COLOR_RED
	case "yellow":
		return uno.COLOR_YELLOW
	case "green":
		return uno.COLOR_GREEN
	case "blue":
		return uno.COLOR_BLUE
	default:
		return uno.COLOR_NONE
	}
}

func (p *unoPlugin) colorCallback(b *seabird.Bot, m *irc.Message) {
	colorStr := m.Trailing()
	if colorStr == "" {
		b.MentionReply(m, "Usage: <prefix>color <color>")
		return
	}

	msg, ok := p.checkGame(b, m, []uno.GameState{uno.STATE_WAITING_COLOR, uno.STATE_WAITING_COLOR_FOUR})
	if !ok {
		b.MentionReply(m, msg)
		return
	}

	color := ColorForString(colorStr)
	if color == uno.COLOR_NONE {
		b.MentionReply(m, "Unknown color \"%s\"", colorStr)
		return
	}

	p.game.ChooseColor(color)
	if p.game.State() == uno.STATE_WAITING_COLOR_FOUR {
		p.game.AdvancePlayer()
		b.Reply(m, "%s forced to draw four cards and skip a turn!", p.game.CurrentPlayer().Name)
		p.game.CurrentPlayer().DrawCards(p.game, 4)
	}
	p.game.AdvancePlayer()
	b.Reply(m, "%s's turn.", p.game.CurrentPlayer().Name)
	b.Reply(m, "%s is on top of discard.", p.game.Discard.Top())
}

func (p *unoPlugin) playCallback(b *seabird.Bot, m *irc.Message) {
	idxStr := m.Trailing()
	if idxStr == "" {
		b.MentionReply(m, "Usage: <prefix>play <hand_index>")
		return
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(p.game.CurrentPlayer().Hand.Cards) {
		b.MentionReply(m, "Bad card index \"%s\"", idxStr)
		return
	}

	msg, ok := p.checkGame(b, m, []uno.GameState{uno.STATE_WAITING_TURN})
	if !ok {
		b.MentionReply(m, msg)
		return
	}

	card, err := p.game.CurrentPlayer().RemoveCard(idx)
	if err != nil {
		b.MentionReply(m, "Bad card index \"%s\"", idxStr)
		return
	}

	b.Reply(m, "Playing %s's %s", p.game.CurrentPlayer().Name, card.String())
	p.game.RunCard(card)
}

func (p *unoPlugin) drawCallback(b *seabird.Bot, m *irc.Message) {
	msg, ok := p.checkGame(b, m, []uno.GameState{uno.STATE_WAITING_TURN})
	if !ok {
		b.MentionReply(m, msg)
		return
	}

	p.sendMessages(b, m, p.game.DrawCard())
}

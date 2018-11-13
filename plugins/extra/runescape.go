package extra

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
)

var runescapeOldSchoolSkillNames = []string{
	"total",
	"attack",
	"defence",
	"strength",
	"hitpoints",
	"ranged",
	"prayer",
	"magic",
	"cooking",
	"woodcutting",
	"fletching",
	"fishing",
	"firemaking",
	"crafting",
	"smithing",
	"mining",
	"herblore",
	"agility",
	"thieving",
	"slayer",
	"farming",
	"runecraft",
	"hunter",
	"construction",
}

type runescapeLevelMetadata struct {
	Rank  int
	Level int
	Exp   int

	Player string
	Skill  string
}

type runescapePlugin struct{}

func init() {
	seabird.RegisterPlugin("runescape", newRunescapePlugin)
}

func newRunescapePlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
	p := &runescapePlugin{}

	cm.Event("rlevel", p.levelCallback, &seabird.HelpInfo{
		Usage:       "<player> <skill>",
		Description: "Returns a player's old school runescape skill level",
	})
	cm.Event("rexp", p.expCallback, &seabird.HelpInfo{
		Usage:       "<player> <skill>",
		Description: "Returns a player's old school runescape skill exp",
	})
	cm.Event("rrank", p.rankCallback, &seabird.HelpInfo{
		Usage:       "<player> <skill>",
		Description: "Returns a player's old school runescape skill rank",
	})

	return nil
}

func (p *runescapePlugin) getPlayerSkills(search string) (*runescapeLevelMetadata, error) {
	args := strings.SplitN(search, " ", 2)
	if len(args) != 2 {
		return nil, errors.New("Wrong number of args")
	}
	player := args[0]
	skill := args[1]

	resp, err := http.Get("https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player=" + player)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data := strings.Split(strings.TrimSpace(string(bytes)), "\n")

	// It's not strictly needed to build all this up, but it may be useful later.
	ret := map[string]runescapeLevelMetadata{}
	if len(data) < len(runescapeOldSchoolSkillNames) {
		return nil, fmt.Errorf("Invalid data")
	}

	for i, name := range runescapeOldSchoolSkillNames {
		line := data[i]
		levelData := strings.Split(line, ",")
		if len(levelData) < 3 {
			return nil, fmt.Errorf("Invalid data")
		}
		rank, err := strconv.Atoi(levelData[0])
		if err != nil {
			return nil, err
		}
		level, err := strconv.Atoi(levelData[1])
		if err != nil {
			return nil, err
		}
		exp, err := strconv.Atoi(levelData[2])
		if err != nil {
			return nil, err
		}
		ret[name] = runescapeLevelMetadata{
			Rank:   rank,
			Level:  level,
			Exp:    exp,
			Player: player,
			Skill:  name,
		}
	}

	// Pull out the proper data
	md, ok := ret[skill]
	if !ok {
		return nil, fmt.Errorf("Unknown skill %q", skill)
	}

	return &md, nil
}

func (p *runescapePlugin) levelCallback(b *seabird.Bot, m *irc.Message) {
	trailing := m.Trailing()
	go func() {
		data, err := p.getPlayerSkills(trailing)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, "%s has level %d %s", data.Player, data.Level, data.Skill)
	}()
}

func (p *runescapePlugin) expCallback(b *seabird.Bot, m *irc.Message) {
	trailing := m.Trailing()
	go func() {
		data, err := p.getPlayerSkills(trailing)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, "%s has %d experience in %s", data.Player, data.Exp, data.Skill)
	}()
}

func (p *runescapePlugin) rankCallback(b *seabird.Bot, m *irc.Message) {
	trailing := m.Trailing()
	go func() {
		data, err := p.getPlayerSkills(trailing)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, "%s has rank %d in %s", data.Player, data.Rank, data.Skill)
	}()
}

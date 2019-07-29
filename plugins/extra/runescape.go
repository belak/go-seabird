// +build ignore

package extra

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
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

var levelRegex = regexp.MustCompile(`(\w{2,}|".+?")\s+(\w+)$`)

func init() {
	seabird.RegisterPlugin("runescape", newRunescapePlugin)
}

func newRunescapeLevelMetadata(name, player, line string) (runescapeLevelMetadata, error) {
	var emptySkill runescapeLevelMetadata

	levelData := strings.Split(line, ",")
	if len(levelData) < 3 {
		return emptySkill, fmt.Errorf("Invalid data")
	}
	rank, err := strconv.Atoi(levelData[0])
	if err != nil {
		return emptySkill, err
	}
	level, err := strconv.Atoi(levelData[1])
	if err != nil {
		return emptySkill, err
	}
	exp, err := strconv.Atoi(levelData[2])
	if err != nil {
		return emptySkill, err
	}
	return runescapeLevelMetadata{
		Rank:   rank,
		Level:  level,
		Exp:    exp,
		Player: player,
		Skill:  name,
	}, nil
}

func newRunescapePlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
	p := &runescapePlugin{}

	cm.Event("rlvl", p.levelCallback, &seabird.HelpInfo{
		Usage:       "<player> <skill>",
		Description: "Returns a player's Old-School Runescape skill level",
	})
	cm.Event("rexp", p.expCallback, &seabird.HelpInfo{
		Usage:       "<player> <skill>",
		Description: "Returns a player's Old-School Runescape skill exp",
	})
	cm.Event("rrank", p.rankCallback, &seabird.HelpInfo{
		Usage:       "<player> <skill>",
		Description: "Returns a player's Old-School Runescape skill rank",
	})

	return nil
}

func getCombatLevel(attack, defence, strength, hitpoints, ranged, prayer, magic int) int {
	// Formula was taken from https://oldschool.runescape.wiki/w/Combat_level#Mathematics
	base := 0.25 * (float64(defence) + float64(hitpoints) + math.Floor(float64(prayer)/2))
	meleeOption := 0.325 * (float64(attack) + float64(strength))
	rangedFloat := float64(ranged)
	rangedOption := 0.325 * (math.Floor(rangedFloat/2) + rangedFloat)
	magicFloat := float64(magic)
	magicOption := 0.325 * (math.Floor(magicFloat/2) + magicFloat)

	return int(math.Floor(base + math.Max(meleeOption, math.Max(rangedOption, magicOption))))
}

func (p *runescapePlugin) getPlayerSkills(search string) (runescapeLevelMetadata, error) {
	var emptySkill runescapeLevelMetadata

	found := false
	player := ""
	skill := ""

	matches := levelRegex.FindAllStringSubmatch(search, -1)
	for _, v := range matches {
		if strings.HasPrefix(v[1], "\"") {
			v[1] = v[1][1 : len(v[1])-1]
		}
		player = v[1]
		skill = v[2]
		found = true
	}

	if !found {
		return emptySkill, errors.New("Unable to parse player or skill")
	}

	resp, err := http.Get("https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player=" + player)
	if err != nil {
		return emptySkill, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return emptySkill, err
	}
	data := strings.Split(strings.TrimSpace(string(bytes)), "\n")

	// It's not strictly needed to build all this up, but it may be useful later.
	var ret = make(map[string]runescapeLevelMetadata)
	if len(data) < len(runescapeOldSchoolSkillNames) {
		return emptySkill, fmt.Errorf("Invalid data")
	}

	for i, name := range runescapeOldSchoolSkillNames {
		md, err := newRunescapeLevelMetadata(name, player, data[i])
		if err != nil {
			return emptySkill, err
		}

		ret[md.Skill] = md
	}

	if skill == "combat" {
		combat := getCombatLevel(
			ret["attack"].Level,
			ret["defence"].Level,
			ret["strength"].Level,
			ret["hitpoints"].Level,
			ret["ranged"].Level,
			ret["prayer"].Level,
			ret["magic"].Level)
		return runescapeLevelMetadata{
			Rank:   -1,
			Level:  combat,
			Exp:    -1,
			Player: player,
			Skill:  skill,
		}, nil
	}

	// Pull out the proper data
	md, ok := ret[skill]
	if !ok {
		return emptySkill, fmt.Errorf("Unknown skill %q", skill)
	}

	return md, nil
}

func (p *runescapePlugin) levelCallback(b *seabird.Bot, m *irc.Message) {
	trailing := strings.ToLower(m.Trailing())
	go func() {
		data, err := p.getPlayerSkills(trailing)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, "%s has level %s %s", data.Player, utils.PrettifyNumber(data.Level), data.Skill)
	}()
}

func (p *runescapePlugin) expCallback(b *seabird.Bot, m *irc.Message) {
	trailing := strings.ToLower(m.Trailing())
	go func() {
		data, err := p.getPlayerSkills(trailing)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, "%s has %s experience in %s", data.Player, utils.PrettifySuffix(data.Exp), data.Skill)
	}()
}

func (p *runescapePlugin) rankCallback(b *seabird.Bot, m *irc.Message) {
	trailing := strings.ToLower(m.Trailing())
	go func() {
		data, err := p.getPlayerSkills(trailing)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, "%s has rank %s in %s", data.Player, utils.PrettifyNumber(data.Rank), data.Skill)
	}()
}

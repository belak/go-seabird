package extra

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
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

var runescapeOldSchoolSkillAliases = map[string]string{
	"overall":      "total",
	"atk":          "attack",
	"att":          "attack",
	"def":          "defence",
	"defense":      "defence",
	"str":          "strength",
	"hp":           "hitpoints",
	"range":        "ranged",
	"ranging":      "ranged",
	"pray":         "prayer",
	"mage":         "magic",
	"cook":         "cooking",
	"wc":           "woodcutting",
	"fletch":       "fletching",
	"fish":         "fishing",
	"fm":           "firemaking",
	"craft":        "crafting",
	"herb":         "herblore",
	"agi":          "agility",
	"farm":         "farming",
	"runecrafting": "runecraft",
	"rc":           "runecraft",
	"con":          "construction",
}

type runescapeLevelMetadata struct {
	Rank  int
	Level int
	Exp   int

	Player string
	Skill  string
}

type runescapePlugin struct{}

var levelRegex = regexp.MustCompile(`(\w{2,}|".+?")\s+((\w+\s*)+)$`)

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

func resolveAlias(possibleAlias string) string {
	if val, ok := runescapeOldSchoolSkillAliases[possibleAlias]; ok {
		return val
	}

	return possibleAlias
}

//nolint:funlen
func (p *runescapePlugin) getPlayerSkills(search string) (map[string]runescapeLevelMetadata, error) {
	var emptySkills map[string]runescapeLevelMetadata

	var (
		found        = false
		player       string
		skillsString string

		skills []string
	)

	matches := levelRegex.FindAllStringSubmatch(search, -1)
	for _, v := range matches {
		if strings.HasPrefix(v[1], "\"") {
			v[1] = v[1][1 : len(v[1])-1]
		}

		player = v[1]
		skillsString = v[2]
		skills = strings.Fields(skillsString)

		for i, skill := range skills {
			skills[i] = resolveAlias(skill)
		}

		found = true
	}

	if !found {
		return emptySkills, errors.New("Unable to parse player or skill")
	}

	resp, err := http.Get("https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player=" + player)
	if err != nil {
		return emptySkills, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return emptySkills, err
	}

	data := strings.Split(strings.TrimSpace(string(bytes)), "\n")

	// It's not strictly needed to build all this up, but it may be useful later.
	var ret = make(map[string]runescapeLevelMetadata)

	if len(data) < len(runescapeOldSchoolSkillNames) {
		return emptySkills, fmt.Errorf("Invalid data")
	}

	for i, name := range runescapeOldSchoolSkillNames {
		md, err := newRunescapeLevelMetadata(name, player, data[i])
		if err != nil {
			return emptySkills, err
		}

		ret[md.Skill] = md
	}

	returnedSkills := make(map[string]runescapeLevelMetadata)

	for _, skill := range skills {
		if skill == "combat" {
			combat := getCombatLevel(
				ret["attack"].Level,
				ret["defence"].Level,
				ret["strength"].Level,
				ret["hitpoints"].Level,
				ret["ranged"].Level,
				ret["prayer"].Level,
				ret["magic"].Level)
			returnedSkills["combat"] = runescapeLevelMetadata{
				Rank:   -1,
				Level:  combat,
				Exp:    -1,
				Player: player,
				Skill:  skill,
			}

			continue
		}

		// Pull out the proper data
		md, ok := ret[skill]
		if !ok {
			return emptySkills, fmt.Errorf("Unknown skill %q", skill)
		}

		returnedSkills[skill] = md
	}

	return returnedSkills, nil
}

func sortedSkillNames(skills map[string]runescapeLevelMetadata) []string {
	var names []string
	for name := range skills {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (p *runescapePlugin) levelCallback(b *seabird.Bot, r *seabird.Request) {
	trailing := strings.ToLower(r.Message.Trailing())

	go func() {
		skills, err := p.getPlayerSkills(trailing)
		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		playerName := ""
		names := sortedSkillNames(skills)

		var responses []string
		var skill runescapeLevelMetadata
		for _, name := range names {
			skill = skills[name]
			playerName = skill.Player
			responses = append(responses, fmt.Sprintf("level %s %s", utils.PrettifyNumber(skill.Level), skill.Skill))
		}

		r.MentionReply("%s has %s", playerName, strings.Join(responses, ", "))
	}()
}

func (p *runescapePlugin) expCallback(b *seabird.Bot, r *seabird.Request) {
	trailing := strings.ToLower(r.Message.Trailing())

	go func() {
		skills, err := p.getPlayerSkills(trailing)
		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		playerName := ""
		names := sortedSkillNames(skills)

		var responses []string
		var skill runescapeLevelMetadata
		for _, name := range names {
			skill = skills[name]
			playerName = skill.Player
			responses = append(responses, fmt.Sprintf("%s experience in %s", utils.PrettifySuffix(skill.Exp), skill.Skill))
		}

		r.MentionReply("%s has %s", playerName, strings.Join(responses, ", "))
	}()
}

func (p *runescapePlugin) rankCallback(b *seabird.Bot, r *seabird.Request) {
	trailing := strings.ToLower(r.Message.Trailing())

	go func() {
		skills, err := p.getPlayerSkills(trailing)
		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		playerName := ""
		names := sortedSkillNames(skills)

		var responses []string
		var skill runescapeLevelMetadata
		for _, name := range names {
			skill = skills[name]
			playerName = skill.Player
			responses = append(responses, fmt.Sprintf("rank %s in %s", utils.PrettifyNumber(skill.Rank), skill.Skill))
		}

		r.MentionReply("%s has %s", playerName, strings.Join(responses, ", "))
	}()
}

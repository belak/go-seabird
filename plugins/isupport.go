package plugins

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/belak/go-seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("isupport", newISupportPlugin)
}

// ISupportPlugin tracks which ISupport features are enabled on the
// current connection.
type ISupportPlugin struct {
	raw map[string]string
}

func newISupportPlugin(b *seabird.Bot, bm *seabird.BasicMux) *ISupportPlugin {
	p := &ISupportPlugin{
		raw: map[string]string{
			"PREFIX": "(ov)@+",
		},
	}
	bm.Event("005", p.handle005)
	return p
}

func (p *ISupportPlugin) handle005(b *seabird.Bot, m *irc.Message) {
	logger := b.GetLogger()

	// Check for really old servers (or servers which based 005 off of rfc2812
	if !strings.HasSuffix(m.Trailing(), "server") {
		logger.Warn("This server doesn't appear to support ISupport messages. Here there be dragons.")
		return
	}

	if len(m.Params) < 2 {
		logger.Warn("Not enough params in ISupport message")
		return
	}

	for _, param := range m.Params[1 : len(m.Params)-1] {
		data := strings.SplitN(param, "=", 2)
		if len(data) < 2 {
			p.raw[data[0]] = ""
			continue
		}

		p.raw[data[0]] = data[1]

		logger.WithFields(logrus.Fields{
			"key": data[0],
			"raw": data[1],
		}).Debug("Setting ISupport value")
	}
}

// IsEnabled will check for boolean ISupport values
func (p *ISupportPlugin) IsEnabled(key string) bool {
	_, ok := p.raw[key]
	return ok
}

// GetList will check for list ISupportValues
func (p *ISupportPlugin) GetList(key string) ([]string, bool) {
	data, ok := p.raw[key]
	if !ok {
		return nil, false
	}

	return strings.Split(data, ","), true
}

// GetMap will check for map ISupport values
func (p *ISupportPlugin) GetMap(key string) (map[string]string, bool) {
	data, ok := p.raw[key]
	if !ok {
		return nil, false
	}

	ret := make(map[string]string)
	for _, v := range strings.Split(data, ",") {
		innerData := strings.SplitN(v, ":", 2)
		if len(innerData) != 2 {
			return nil, false
		}

		ret[innerData[0]] = innerData[1]
	}

	return ret, true
}

// GetRaw will get the raw ISupport values
func (p *ISupportPlugin) GetRaw(key string) (string, bool) {
	ret, ok := p.raw[key]
	return ret, ok
}

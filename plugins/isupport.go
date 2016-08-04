package plugins

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/belak/go-seabird/seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("isupport", newISupportPlugin)
}

// ISupportPlugin tracks which ISupport features are enabled on the
// current connection.
type ISupportPlugin struct {
	enabled  map[string]struct{}
	mapData  map[string]map[string]string
	listData map[string][]string
	values   map[string]string
	raw      map[string]string
}

func newISupportPlugin(b *seabird.Bot, bm *seabird.BasicMux) *ISupportPlugin {
	p := &ISupportPlugin{
		make(map[string]struct{}),
		make(map[string]map[string]string),
		make(map[string][]string),
		make(map[string]string),
		make(map[string]string),
	}
	bm.Event("005", p.handle005)
	return p
}

func (p *ISupportPlugin) handle005(b *seabird.Bot, m *irc.Message) {
	rawLogger := b.GetLogger()

	// Check for really old servers (or servers which based 005 off of rfc2812
	if !strings.HasSuffix(m.Trailing(), "server") {
		return
	}

	if len(m.Params) < 2 {
		return
	}

	for _, param := range m.Params[1 : len(m.Params)-1] {
		data := strings.SplitN(param, "=", 2)
		if len(data) < 2 {
			rawLogger.WithField("key", data[0]).Debug("Setting ISupport enabled")
			p.enabled[data[0]] = struct{}{}
			p.raw[data[0]] = ""
			continue
		}

		p.raw[data[0]] = data[1]

		logger := rawLogger.WithFields(logrus.Fields{
			"key": data[0],
			"raw": data[1],
		})

		retList := []string{}
		retMap := map[string]string{}
		for _, v := range strings.Split(data[1], ",") {
			innerData := strings.SplitN(v, ":", 2)
			if len(innerData) < 2 {
				retList = append(retList, innerData[0])
			} else {
				retMap[innerData[0]] = innerData[1]
			}
		}

		if len(retList) != 0 && len(retMap) != 0 {
			logger.Warn("ISupport key contains both a list and a map")
			fmt.Printf("Key %s contains both a list and a map\n", data[0])
		}

		if len(retList) > 0 {
			if len(retList) == 1 {
				p.values[data[0]] = retList[0]
				logger.WithField("val", retList[0]).Debug("Setting ISupport value")
			} else {
				p.listData[data[0]] = retList
				logger.WithField("val", retList).Debug("Setting ISupport list")
			}
		}

		if len(retMap) > 0 {
			p.mapData[data[0]] = retMap
			logger.WithField("val", retMap).Debug("Setting ISupport map")
		}

		if len(retList) == 0 && len(retMap) == 0 {
			p.enabled[data[0]] = struct{}{}
			logger.Debug("Setting ISupport enabled")
		}
	}
}

// IsEnabled will check for boolean ISupport values
func (p *ISupportPlugin) IsEnabled(key string) bool {
	_, ok := p.enabled[key]
	return ok
}

// GetList will check for list ISupportValues
func (p *ISupportPlugin) GetList(key string) ([]string, bool) {
	ret, ok := p.listData[key]
	return ret, ok
}

// GetMap will check for map ISupport values
func (p *ISupportPlugin) GetMap(key string) (map[string]string, bool) {
	ret, ok := p.mapData[key]
	return ret, ok
}

// GetRaw will get the raw ISupport values
func (p *ISupportPlugin) GetRaw(key string) (string, bool) {
	ret, ok := p.raw[key]
	return ret, ok
}

// GetValue will check for simple ISupport values (lists with 1 element)
func (p *ISupportPlugin) GetValue(key string) (string, bool) {
	ret, ok := p.values[key]
	return ret, ok
}

package url

import (
	"context"
	"net/url"
	"regexp"
	"text/template"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/Sirupsen/logrus"
	"github.com/zmb3/spotify"

	"github.com/belak/go-seabird"
	"github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("url/spotify", newSpotifyProvider)
}

type spotifyConfig struct {
	ClientID     string
	ClientSecret string
}

type spotifyProvider struct {
	api spotify.Client
}

var spotifyPrefix = "[Spotify]"

type spotifyMatch struct {
	regex    *regexp.Regexp
	template *template.Template
	lookup   func(*spotifyProvider, *logrus.Entry, spotify.ID) interface{}
}

var spotifyMatchers = []spotifyMatch{
	{
		regex: regexp.MustCompile(`^/album/(.+)$`),
		template: TemplateMustCompile("spotifyAlbum", `
			{{- .Name }} by
			{{- range $index, $element := .Artists }}
			{{- if $index }},{{ end }} {{ $element.Name -}}
			{{- end }}`),
		lookup: func(s *spotifyProvider, logger *logrus.Entry, id spotify.ID) interface{} {
			album, err := s.api.GetAlbum(id)
			if err != nil {
				logger.WithError(err).Error("Failed to get album info from Spotify")
				return nil
			}
			return album
		},
	},
}

var spotifyAlbumRegex = regexp.MustCompile(`^/album/(.+)$`)

func newSpotifyProvider(b *seabird.Bot, urlPlugin *Plugin) error {
	s := &spotifyProvider{}

	sc := &spotifyConfig{}
	err := b.Config("spotify", sc)
	if err != nil {
		return err
	}

	config := &clientcredentials.Config{
		ClientID:     sc.ClientID,
		ClientSecret: sc.ClientSecret,
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		return err
	}

	s.api = spotify.Authenticator{}.NewClient(token)

	urlPlugin.RegisterProvider("open.spotify.com", s.Handle)

	return nil
}

func (s *spotifyProvider) Handle(b *seabird.Bot, m *irc.Message, u *url.URL) bool {
	logger := b.GetLogger()

	for _, matcher := range spotifyMatchers {
		if !matcher.regex.MatchString(u.Path) {
			continue
		}

		matches := matcher.regex.FindStringSubmatch(u.Path)
		if len(matches) != 2 {
			return false
		}

		data := matcher.lookup(s, logger, spotify.ID(matches[1]))
		if data == nil {
			return false
		}

		return RenderRespond(b, m, logger, matcher.template, spotifyPrefix, data)
	}

	return false
}

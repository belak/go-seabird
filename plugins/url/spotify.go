package url

import (
	"context"
	"net/url"
	"regexp"
	"text/template"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
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
	matchCount int
	regex      *regexp.Regexp
	uriRegex   *regexp.Regexp
	template   *template.Template
	lookup     func(*spotifyProvider, *logrus.Entry, []string) interface{}
}

var spotifyMatchers = []spotifyMatch{
	{
		matchCount: 1,
		regex:      regexp.MustCompile(`^/artist/(.+)$`),
		uriRegex:   regexp.MustCompile(`\bspotify:artist:(\w+)\b`),
		template:   TemplateMustCompile("spotifyArtist", `{{- .Name -}}`),
		lookup: func(s *spotifyProvider, logger *logrus.Entry, matches []string) interface{} {
			artist, err := s.api.GetArtist(spotify.ID(matches[0]))
			if err != nil {
				logger.WithError(err).Error("Failed to get artist info from Spotify")
				return nil
			}
			return artist
		},
	},
	{
		matchCount: 1,
		regex:      regexp.MustCompile(`^/album/(.+)$`),
		uriRegex:   regexp.MustCompile(`\bspotify:album:(\w+)\b`),
		template: TemplateMustCompile("spotifyAlbum", `
			{{- .Name }} by
			{{- range $index, $element := .Artists }}
			{{- if $index }},{{ end }} {{ $element.Name -}}
			{{- end }} ({{ .Tracks.Total }} {{ pluralize .Tracks.Total "track" }})`),
		lookup: func(s *spotifyProvider, logger *logrus.Entry, matches []string) interface{} {
			album, err := s.api.GetAlbum(spotify.ID(matches[0]))
			if err != nil {
				logger.WithError(err).Error("Failed to get album info from Spotify")
				return nil
			}
			return album
		},
	},
	{
		matchCount: 1,
		regex:      regexp.MustCompile(`^/track/(.+)$`),
		uriRegex:   regexp.MustCompile(`\bspotify:track:(\w+)\b`),
		template: TemplateMustCompile("spotifyTrack", `
			"{{ .Name }}" from {{ .Album.Name }} by
			{{- range $index, $element := .Artists }}
			{{- if $index }},{{ end }} {{ $element.Name }}
			{{- end }}`),
		lookup: func(s *spotifyProvider, logger *logrus.Entry, matches []string) interface{} {
			track, err := s.api.GetTrack(spotify.ID(matches[0]))
			if err != nil {
				logger.WithError(err).Error("Failed to get track info from Spotify")
				return nil
			}
			return track
		},
	},
	{
		matchCount: 2,
		regex:      regexp.MustCompile(`^/user/([^/]*)/playlist/([^/]*)$`),
		uriRegex:   regexp.MustCompile(`\bspotify:user:(\w+):playlist:(\w+)\b`),
		template: TemplateMustCompile("spotifyPlaylist", `
			"{{- .Name }}" playlist by {{ .Owner.DisplayName }} ({{ .Tracks.Total }} {{ pluralize .Tracks.Total "track" }})`),
		lookup: func(s *spotifyProvider, logger *logrus.Entry, matches []string) interface{} {
			playlist, err := s.api.GetPlaylist(matches[0], spotify.ID(matches[1]))
			if err != nil {
				logger.WithError(err).Error("Failed to get track info from Spotify")
				return nil
			}
			return playlist
		},
	},
}

func newSpotifyProvider(b *seabird.Bot, m *seabird.BasicMux, urlPlugin *Plugin) error {
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

	m.Event("PRIVMSG", s.privmsgCallback)

	urlPlugin.RegisterProvider("open.spotify.com", s.HandleURL)

	return nil
}

func (s *spotifyProvider) privmsgCallback(b *seabird.Bot, m *irc.Message) {
	for _, matcher := range spotifyMatchers {
		if s.handleTarget(b, m, matcher, matcher.uriRegex, m.Trailing()) {
			return
		}
	}
}

func (s *spotifyProvider) HandleURL(b *seabird.Bot, m *irc.Message, u *url.URL) bool {
	for _, matcher := range spotifyMatchers {
		if s.handleTarget(b, m, matcher, matcher.regex, u.Path) {
			return true
		}
	}
	return false
}

func (s *spotifyProvider) handleTarget(b *seabird.Bot, m *irc.Message, matcher spotifyMatch, regex *regexp.Regexp, target string) bool {
	logger := b.GetLogger()

	if !regex.MatchString(target) {
		return false
	}

	matches := regex.FindStringSubmatch(target)
	if len(matches) != matcher.matchCount+1 {
		return false
	}

	data := matcher.lookup(s, logger, matches[1:])
	if data == nil {
		return false
	}

	return RenderRespond(b, m, logger, matcher.template, spotifyPrefix, data)
}

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
	matchCount int
	regex      *regexp.Regexp
	template   *template.Template
	lookup     func(*spotifyProvider, *logrus.Entry, []string) interface{}
}

var spotifyMatchers = []spotifyMatch{
	{
		matchCount: 1,
		regex:      regexp.MustCompile(`^/artist/(.+)$`),
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
		if len(matches) != matcher.matchCount+1 {
			return false
		}

		data := matcher.lookup(s, logger, matches[1:])
		if data == nil {
			return false
		}

		return RenderRespond(b, m, logger, matcher.template, spotifyPrefix, data)
	}

	return false
}

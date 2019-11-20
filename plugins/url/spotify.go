package url

import (
	"context"
	"net/url"
	"regexp"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
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
	uriRegex *regexp.Regexp
	template *template.Template
	lookup   func(*spotifyProvider, *logrus.Entry, []string) interface{}
}

var spotifyMatchers = []spotifyMatch{
	{
		regex:    regexp.MustCompile(`^/artist/(.+)$`),
		uriRegex: regexp.MustCompile(`\bspotify:artist:(\w+)\b`),
		template: internal.TemplateMustCompile("spotifyArtist", `{{- .Name -}}`),
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
		regex:    regexp.MustCompile(`^/album/(.+)$`),
		uriRegex: regexp.MustCompile(`\bspotify:album:(\w+)\b`),
		template: internal.TemplateMustCompile("spotifyAlbum", `
			{{- .Name }} by
			{{- range $index, $element := .Artists }}
			{{- if $index }},{{ end }} {{ $element.Name -}}
			{{- end }} ({{ pluralize .Tracks.Total "track" }})`),
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
		regex:    regexp.MustCompile(`^/track/(.+)$`),
		uriRegex: regexp.MustCompile(`\bspotify:track:(\w+)\b`),
		template: internal.TemplateMustCompile("spotifyTrack", `
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
		regex:    regexp.MustCompile(`^/playlist/([^/]*)$`),
		uriRegex: regexp.MustCompile(`\bspotify:playlist:(\w+)\b`),
		template: internal.TemplateMustCompile("spotifyPlaylist", `
			"{{- .Name }}" playlist by {{ .Owner.DisplayName }} ({{ pluralize .Tracks.Total "track" }})`),
		lookup: func(s *spotifyProvider, logger *logrus.Entry, matches []string) interface{} {
			playlist, err := s.api.GetPlaylist(spotify.ID(matches[0]))
			if err != nil {
				logger.WithError(err).Error("Failed to get track info from Spotify")
				return nil
			}
			return playlist
		},
	},
}

func newSpotifyProvider(b *seabird.Bot) error {
	if err := b.EnsurePlugin("url"); err != nil {
		return err
	}

	bm := b.BasicMux()
	urlPlugin := CtxPlugin(b.Context())

	s := &spotifyProvider{}

	sc := &spotifyConfig{}
	if err := b.Config("spotify", sc); err != nil {
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

	bm.Event("PRIVMSG", s.privmsgCallback)

	urlPlugin.RegisterProvider("open.spotify.com", s.HandleURL)

	return nil
}

func (s *spotifyProvider) privmsgCallback(r *seabird.Request) {
	for _, matcher := range spotifyMatchers {
		if s.handleTarget(r, matcher, matcher.uriRegex, r.Message.Trailing()) {
			return
		}
	}
}

func (s *spotifyProvider) HandleURL(r *seabird.Request, u *url.URL) bool {
	for _, matcher := range spotifyMatchers {
		if s.handleTarget(r, matcher, matcher.regex, u.Path) {
			return true
		}
	}

	return false
}

func (s *spotifyProvider) handleTarget(r *seabird.Request, matcher spotifyMatch, regex *regexp.Regexp, target string) bool {
	logger := r.GetLogger("url/spotify")

	if !regex.MatchString(target) {
		return false
	}

	matches := regex.FindStringSubmatch(target)
	if len(matches) != 2 {
		return false
	}

	data := matcher.lookup(s, logger, matches[1:])
	if data == nil {
		return false
	}

	return internal.RenderRespond(r.Reply, logger, matcher.template, spotifyPrefix, data)
}

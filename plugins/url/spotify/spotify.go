package spotify

import (
	"context"
	"net/url"
	"regexp"
	"sync"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
	urlPlugin "github.com/belak/go-seabird/plugins/url"
)

func init() {
	seabird.RegisterPlugin("url/spotify", newSpotifyProvider)
}

type spotifyConfig struct {
	ClientID     string
	ClientSecret string
}

type spotifyProvider struct {
	lock   *sync.RWMutex
	config *clientcredentials.Config
	token  *oauth2.Token
}

var spotifyPrefix = "[Spotify]"

type spotifyMatch struct {
	regex    *regexp.Regexp
	uriRegex *regexp.Regexp
	template *template.Template
	lookup   func(spotify.Client, *logrus.Entry, []string) interface{}
}

var spotifyMatchers = []spotifyMatch{
	{
		regex:    regexp.MustCompile(`^/artist/(.+)$`),
		uriRegex: regexp.MustCompile(`\bspotify:artist:(\w+)\b`),
		template: internal.TemplateMustCompile("spotifyArtist", `{{- .Name -}}`),
		lookup: func(api spotify.Client, logger *logrus.Entry, matches []string) interface{} {
			artist, err := api.GetArtist(spotify.ID(matches[0]))
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
		lookup: func(api spotify.Client, logger *logrus.Entry, matches []string) interface{} {
			album, err := api.GetAlbum(spotify.ID(matches[0]))
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
		lookup: func(api spotify.Client, logger *logrus.Entry, matches []string) interface{} {
			track, err := api.GetTrack(spotify.ID(matches[0]))
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
		lookup: func(api spotify.Client, logger *logrus.Entry, matches []string) interface{} {
			playlist, err := api.GetPlaylist(spotify.ID(matches[0]))
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
	urlPlugin := urlPlugin.CtxPlugin(b.Context())

	s := &spotifyProvider{
		lock: &sync.RWMutex{},
	}

	sc := &spotifyConfig{}
	if err := b.Config("spotify", sc); err != nil {
		return err
	}

	s.config = &clientcredentials.Config{
		ClientID:     sc.ClientID,
		ClientSecret: sc.ClientSecret,
		TokenURL:     spotify.TokenURL,
	}

	// Ensure we have valid credentials
	_, err := s.getAPI()
	if err != nil {
		return err
	}

	bm.Event("PRIVMSG", s.privmsgCallback)

	urlPlugin.RegisterProvider("open.spotify.com", s.HandleURL)

	return nil
}

func (s *spotifyProvider) getAPI() (spotify.Client, error) {
	// If we already have a valid token, we can bail
	s.lock.RLock()
	if s.token != nil && s.token.Valid() {
		s.lock.RUnlock()
		return spotify.Authenticator{}.NewClient(s.token), nil
	}
	s.lock.RUnlock()

	s.lock.Lock()
	defer s.lock.Unlock()

	token, err := s.config.Token(context.Background())
	if err != nil {
		return spotify.Client{}, err
	}

	s.token = token

	return spotify.Authenticator{}.NewClient(s.token), nil
}

func (s *spotifyProvider) privmsgCallback(r *seabird.Request) {
	logger := r.GetLogger("url/spotify")

	api, err := s.getAPI()
	if err != nil {
		logger.WithError(err).Error("Failed to get token from Spotify")
		return
	}

	for _, matcher := range spotifyMatchers {
		if s.handleTarget(r, api, logger, matcher, matcher.uriRegex, r.Message.Trailing()) {
			return
		}
	}
}

func (s *spotifyProvider) HandleURL(r *seabird.Request, u *url.URL) bool {
	logger := r.GetLogger("url/spotify")

	api, err := s.getAPI()
	if err != nil {
		logger.WithError(err).Error("Failed to get token from Spotify")
		return false
	}

	for _, matcher := range spotifyMatchers {
		if s.handleTarget(r, api, logger, matcher, matcher.regex, u.Path) {
			return true
		}
	}

	return false
}

func (s *spotifyProvider) handleTarget(r *seabird.Request, api spotify.Client, logger *logrus.Entry, matcher spotifyMatch, regex *regexp.Regexp, target string) bool {
	if !regex.MatchString(target) {
		return false
	}

	matches := regex.FindStringSubmatch(target)
	if len(matches) != 2 {
		return false
	}

	data := matcher.lookup(api, logger, matches[1:])
	if data == nil {
		return false
	}

	return internal.RenderRespond(r.Replyf, logger, matcher.template, spotifyPrefix, data)
}

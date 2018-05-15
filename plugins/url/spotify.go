package url

import (
	"context"
	"net/url"
	"regexp"

	"golang.org/x/oauth2/clientcredentials"

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
var spotifyAlbumRegex = regexp.MustCompile(`^/album/(.+)$`)

var spotifyAlbumTemplate = TemplateMustCompile("spotifyAlbum", `
{{- .album.Name }} by
{{- range $index, $element := .album.Artists }}
	{{- if $index }},{{ end }} {{ $element.Name -}}
{{- end }}
`)

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
	if spotifyAlbumRegex.MatchString(u.Path) {
		return s.getAlbum(b, m, u.Path)
	}
	return false
}

func (s *spotifyProvider) getAlbum(b *seabird.Bot, m *irc.Message, url string) bool {
	logger := b.GetLogger()

	matches := spotifyAlbumRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	album, err := s.api.GetAlbum(spotify.ID(matches[1]))
	if err != nil {
		logger.WithError(err).Error("Failed to get album from Spotify")
		return false
	}

	return RenderRespond(
		b, m, logger, spotifyAlbumTemplate, spotifyPrefix,
		map[string]interface{}{
			"album": album,
		},
	)
}

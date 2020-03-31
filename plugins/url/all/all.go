package all

import (
	// This package is used as a meta-import for all url plugins.
	_ "github.com/belak/go-seabird/plugins/url"
	_ "github.com/belak/go-seabird/plugins/url/bitbucket"
	_ "github.com/belak/go-seabird/plugins/url/github"
	_ "github.com/belak/go-seabird/plugins/url/reddit"
	_ "github.com/belak/go-seabird/plugins/url/spotify"
	_ "github.com/belak/go-seabird/plugins/url/twitter"
	_ "github.com/belak/go-seabird/plugins/url/xkcd"
	_ "github.com/belak/go-seabird/plugins/url/youtube"
)

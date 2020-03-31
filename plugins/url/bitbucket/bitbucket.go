package bitbucket

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
	urlPlugin "github.com/belak/go-seabird/plugins/url"
)

func init() {
	seabird.RegisterPlugin("url/bitbucket", newBitbucketProvider)
}

type bitbucketUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type bitbucketRepo struct {
	Scm         string `json:"scm"`
	Description string `json:"description"`
	FullName    string `json:"full_name"`
	Language    string `json:"language"`
	UpdatedOn   string `json:"updated_on"`
}

type bitbucketIssue struct {
	Status       string        `json:"status"`
	Priority     string        `json:"priority"`
	Title        string        `json:"title"`
	ReportedBy   bitbucketUser `json:"reported_by"`
	CommentCount int           `json:"comment_count"`
	CreatedOn    string        `json:"created_on"`
	Metadata     struct {
		Kind string `json:"kind"`
	} `json:"metadata"`
}

type bitbucketPullRequest struct {
	State        string        `json:"state"`
	Title        string        `json:"title"`
	Author       bitbucketUser `json:"author"`
	CommentCount int           `json:"comment_count"`
	CreatedOn    string        `json:"created_on"`
}

var (
	bitbucketUserRegex  = regexp.MustCompile(`^/([^/]+)$`)
	bitbucketRepoRegex  = regexp.MustCompile(`^/([^/]+)/([^/]+)$`)
	bitbucketIssueRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)/issue/([^/]+)/[^/]+$`)
	bitbucketPullRegex  = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull-request/([^/]+)/.*$`)

	bitbucketPrefix = "[Bitbucket]"

	userURL             = "https://bitbucket.org/api/2.0/users/%s"
	repoURL             = "https://bitbucket.org/api/2.0/repositories/%s/%s"
	repoIssuesURL       = "https://bitbucket.org/api/1.0/repositories/%s/%s/issues/%s"
	repoPullRequestsURL = "https://bitbucket.org/api/2.0/repositories/%s/%s/pullrequests/%s"
)

func newBitbucketProvider(b *seabird.Bot) error {
	err := b.EnsurePlugin("url")
	if err != nil {
		return err
	}

	urlPlugin := urlPlugin.CtxPlugin(b.Context())

	urlPlugin.RegisterProvider("bitbucket.org", bitbucketCallback)

	return nil
}

func bitbucketCallback(r *seabird.Request, url *url.URL) bool {
	//nolint:gocritic
	if bitbucketUserRegex.MatchString(url.Path) {
		return bitbucketGetUser(r, url)
	} else if bitbucketRepoRegex.MatchString(url.Path) {
		return bitbucketGetRepo(r, url)
	} else if bitbucketIssueRegex.MatchString(url.Path) {
		return bitbucketGetIssue(r, url)
	} else if bitbucketPullRegex.MatchString(url.Path) {
		return bitbucketGetPull(r, url)
	}

	return false
}

func bitbucketGetUser(r *seabird.Request, url *url.URL) bool {
	matches := bitbucketUserRegex.FindStringSubmatch(url.Path)
	if len(matches) != 2 {
		return false
	}

	user := matches[1]

	bu := &bitbucketUser{}
	if err := internal.GetJSON(fmt.Sprintf(userURL, user), bu); err != nil {
		return false
	}

	// Jay Vana @jsvana
	r.Replyf("%s %s (@%s)", bitbucketPrefix, bu.DisplayName, bu.Username)

	return true
}

func bitbucketGetRepo(r *seabird.Request, url *url.URL) bool {
	matches := bitbucketRepoRegex.FindStringSubmatch(url.Path)
	if len(matches) != 3 {
		return false
	}

	user := matches[1]
	repo := matches[2]

	br := &bitbucketRepo{}
	if err := internal.GetJSON(fmt.Sprintf(repoURL, user, repo), br); err != nil {
		return false
	}

	// chriskempson/base16-iterm2 [Shell] Last pushed to 15 Nov 2014 - Base16 for iTerm2
	out := br.FullName
	if br.Language != "" {
		out += " [" + br.Language + "]"
	}

	tm, err := time.Parse(time.RFC3339, br.UpdatedOn)
	if err != nil {
		return false
	}

	out += " Last pushed to " + tm.Format("2 Jan 2006")

	r.Replyf("%s %s", bitbucketPrefix, out)

	return true
}

func bitbucketGetIssue(r *seabird.Request, url *url.URL) bool {
	matches := bitbucketIssueRegex.FindStringSubmatch(url.Path)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	issueNum := matches[3]

	bi := &bitbucketIssue{}
	if err := internal.GetJSON(fmt.Sprintf(repoIssuesURL, user, repo, issueNum), bi); err != nil {
		return false
	}

	// If there isn't a user, we can probably assume they're anonymous
	if bi.ReportedBy.Username == "" {
		bi.ReportedBy.Username = "Anonymous"
	}

	// Issue #51 on belak/go-seabird [open] - Expand issues plugin with more of Bitbucket [created 3 Jan 2015]
	out := fmt.Sprintf("Issue #%s on %s/%s [%s]", issueNum, user, repo, bi.Status)
	if bi.Priority != "" && bi.Metadata.Kind != "" {
		out += " [" + bi.Priority + " - " + bi.Metadata.Kind + "]"
	}

	out += " by " + bi.ReportedBy.Username

	if bi.Title != "" {
		out += " - " + bi.Title
	}

	tm, err := time.Parse("2006-01-02T15:04:05.000", bi.CreatedOn)
	if err != nil {
		return false
	}

	out += " [created " + tm.Format("2 Jan 2006") + "]"
	r.Replyf("%s %s", bitbucketPrefix, out)

	return true
}

func bitbucketGetPull(r *seabird.Request, url *url.URL) bool {
	matches := bitbucketPullRegex.FindStringSubmatch(url.Path)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	pullNum := matches[3]

	bpr := &bitbucketPullRequest{}
	if err := internal.GetJSON(fmt.Sprintf(repoPullRequestsURL, user, repo, pullNum), bpr); err != nil {
		return false
	}

	// Pull request #59 on belak/go-seabird created by jsvana [open] - Add stuff to links [created 4 Jan 2015]
	out := fmt.Sprintf("Pull request #%s on %s/%s created by %s [%s]", pullNum, user, repo, bpr.Author.Username, strings.ToLower(bpr.State))
	if bpr.Title != "" {
		out += " - " + bpr.Title
	}

	tm, err := time.Parse("2006-01-02T15:04:05.000000-07:00", bpr.CreatedOn)
	if err != nil {
		return false
	}

	out += " [created " + tm.Format("2 Jan 2006") + "]"

	r.Replyf("%s %s", bitbucketPrefix, out)

	return true
}

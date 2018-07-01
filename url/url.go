package url

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// ErrInvalidURL is returned from Parse when the url is not a valid git url
var ErrInvalidURL = errors.New("Invalid URL")

var gitURLParser = regexp.MustCompile("^git@([\\w\\.]+):(.+)(?:\\.git)?$")
var gitPathParser = regexp.MustCompile("^/?(.+?)/(.+?)(?:\\.git)?$")

// Transport constants
const (
	GitSSHTransport  = "ssh"
	GitHTTPTransport = "http"
)

// GitURL is a url that points to a git repo
type GitURL struct {
	URI       string
	Transport string
	Username  string
	Password  string
	Domain    string
	Owner     string
	Name      string
}

func (g GitURL) String() string {
	return fmt.Sprintf("%s/%s/%s", g.Domain, g.Owner, g.Name)
}

// Parse gets a url string and returns a URL object, or an error
func Parse(uri string) (GitURL, error) {
	if uri == "" {
		return GitURL{}, ErrInvalidURL
	}
	if strings.HasPrefix(uri, "git@") {
		return parseGitSchemaURL(uri)
	}
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return parseHTTPSchema(uri)
	}
	return GitURL{}, ErrInvalidURL
}

// ToPath creates a path with the domain, owner and name
func (g GitURL) ToPath() string {
	return path.Join(g.Domain, g.Owner, g.Name)
}

func parseGitSchemaURL(uri string) (GitURL, error) {
	if !gitURLParser.MatchString(uri) {
		return GitURL{}, ErrInvalidURL
	}

	matches := gitURLParser.FindStringSubmatch(uri)
	if len(matches) != 3 {
		return GitURL{}, ErrInvalidURL
	}

	domain := matches[1]
	path := matches[2]

	owner, name, err := parsePath(path)
	if err != nil {
		return GitURL{}, err
	}

	return GitURL{
		Transport: GitSSHTransport,
		URI:       uri,
		Domain:    domain,
		Username:  "git",
		Owner:     owner,
		Name:      name,
	}, nil
}

func parseHTTPSchema(uri string) (GitURL, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return GitURL{}, err
	}

	owner, name, err := parsePath(u.Path)
	if err != nil {
		return GitURL{}, err
	}

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	return GitURL{
		Transport: GitHTTPTransport,
		URI:       uri,
		Domain:    u.Hostname(),
		Username:  username,
		Password:  password,
		Owner:     owner,
		Name:      name,
	}, nil

}

func parsePath(path string) (string, string, error) {
	matches := gitPathParser.FindStringSubmatch(path)
	if len(matches) != 3 {
		return "", "", ErrInvalidURL
	}

	return matches[1], matches[2], nil
}

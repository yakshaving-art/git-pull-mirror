package git

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

// GitURL is a url that points to a git repo
type GitURL struct {
	URI    string
	Domain string
	Owner  string
	Name   string
}

func (g GitURL) String() string {
	return fmt.Sprintf("%s/%s/%s", g.Domain, g.g.Owner, g.Name)
}

// Parse gets a url string and returns a URL object, or an error
func Parse(uri string) (*GitURL, error) {
	if uri == "" {
		return nil, ErrInvalidURL
	}
	if strings.HasPrefix(uri, "git@") {
		return parseGitSchemaURL(uri)
	}
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return parseHTTPSchema(uri)
	}
	return nil, ErrInvalidURL
}

// ToPath creates a path with the domain, owner and name
func (g GitURL) ToPath() string {
	return path.Join(g.Domain, g.Owner, g.Name)
}

func parseGitSchemaURL(uri string) (*GitURL, error) {
	if !gitURLParser.MatchString(uri) {
		return nil, ErrInvalidURL
	}

	matches := gitURLParser.FindStringSubmatch(uri)
	if len(matches) != 3 {
		return nil, ErrInvalidURL
	}
	fmt.Printf("Found matches %#v\n", matches)

	domain := matches[1]
	path := matches[2]

	owner, name, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found owner %s, name %s\n", owner, name)

	return &GitURL{
		URI:    uri,
		Domain: domain,
		Owner:  owner,
		Name:   name,
	}, nil
}

func parseHTTPSchema(uri string) (*GitURL, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	owner, name, err := parsePath(u.Path)
	if err != nil {
		return nil, err
	}

	return &GitURL{
		URI:    uri,
		Domain: u.Hostname(),
		Owner:  owner,
		Name:   name,
	}, nil

}

func parsePath(path string) (string, string, error) {
	matches := gitPathParser.FindStringSubmatch(path)
	if len(matches) != 3 {
		return "", "", ErrInvalidURL
	}

	return matches[1], matches[2], nil
}

package git

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
)

// Client is a git client that is used to handle repositories
type Client struct {
	localPath string
}

// NewClient returns a new git client
func NewClient(path string) Client {
	return Client{
		localPath: path,
	}
}

// CloneOrOpen ensures that the repo exists in the indicated path
func (c Client) CloneOrOpen(g *GitURL) (Repository, error) {
	r, err := git.PlainOpen(g.ToPath())
	if err == git.ErrRepositoryNotExists {
		logrus.Debugf("Could not find repository %s, cloning into %s", g, g.ToPath())

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		r, err := git.PlainCloneContext(ctx, g.ToPath(), true, &git.CloneOptions{
			URL:          g.URI,
			SingleBranch: false,
			RemoteName:   "origin",
		})
	}
	if err != nil {
		return Repository{}, fmt.Errorf("failed to clone or open repo %s: %s", g, err)
	}
	return Repository{
		url:     g,
		gitRepo: r,
	}, nil
}

// Repository is a git repo that enables to pull and push. Having an instance of this object means that we have a valid repo
type Repository struct {
	url     *GitURL
	gitRepo *git.Repository
}

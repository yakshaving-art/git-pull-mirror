package server

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
)

// Repository is a git repo that enables to pull and push. Having an instance of this object means that we have a valid repo
type Repository struct {
	url     url.GitURL
	gitRepo *git.Repository
}

// Remotes names
const (
	OriginRemote = "origin"
	TargetRemote = "target"
)

type gitClient struct {
	localPath string
}

func newClient(path string) gitClient {
	return gitClient{
		localPath: path,
	}
}

func (g gitClient) PathFor(origin url.GitURL) string {
	return filepath.Join(g.localPath, origin.ToPath())
}

// CloneOrOpen ensures that the repo exists in the indicated path
func (g gitClient) CloneOrOpen(origin url.GitURL, target string) (Repository, error) {
	r, err := git.PlainOpen(g.PathFor(origin))
	if err == git.ErrRepositoryNotExists {
		logrus.Debugf("Could not find repository %s, cloning into %s", origin, g.PathFor(origin))

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		r, err = git.PlainCloneContext(ctx, g.PathFor(origin), true, &git.CloneOptions{
			URL:          origin.URI,
			SingleBranch: false,
			RemoteName:   OriginRemote,
		})
	}

	if err != nil {
		return Repository{}, fmt.Errorf("failed to clone or open repo %s: %s", g, err)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: TargetRemote,
		URLs: []string{target},
	})
	if err != nil {
		return Repository{}, fmt.Errorf("failed to add target remote %s to repo %s: %s", target, g, err)
	}

	return Repository{
		url:     origin,
		gitRepo: r,
	}, nil
}

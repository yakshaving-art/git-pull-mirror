package server

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
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
	localPath      string
	timeoutSeconds time.Duration

	repositories []Repository
	wg           *sync.WaitGroup
}

func newClient(localPath string, timeoutSeconds int) gitClient {
	return gitClient{
		localPath:      localPath,
		timeoutSeconds: time.Duration(timeoutSeconds) * time.Second,
	}
}

func (g gitClient) pathFor(origin url.GitURL) string {
	return filepath.Join(g.localPath, origin.ToPath())
}

// CloneOrOpen ensures that the repo exists in the indicated path
func (g gitClient) CloneOrOpen(origin url.GitURL, target string) (Repository, error) {
	r, err := git.PlainOpen(g.pathFor(origin))
	if err == git.ErrRepositoryNotExists {
		logrus.Debugf("Could not find repository %s, cloning into %s", origin, g.pathFor(origin))

		ctx, cancel := context.WithTimeout(context.Background(), g.timeoutSeconds*time.Second)
		defer cancel()

		r, err = git.PlainCloneContext(ctx, g.pathFor(origin), true, &git.CloneOptions{
			URL:          origin.URI,
			SingleBranch: false,
			RemoteName:   OriginRemote,
		})
		if err != nil {
			return Repository{}, fmt.Errorf("failed to clone %s: %s", origin, err)
		}

		_, err = r.CreateRemote(&config.RemoteConfig{
			Name: TargetRemote,
			URLs: []string{target},
		})
		if err != nil {
			return Repository{}, fmt.Errorf("failed to add target remote %s to repo %s: %s", target, origin, err)
		}

	} else if err != nil {
		return Repository{}, fmt.Errorf("failed to open repo %s: %s", origin, err)
	} else {
		logrus.Debugf("repository %s already exists locally", origin)
	}

	return Repository{
		url:     origin,
		gitRepo: r,
	}, nil
}

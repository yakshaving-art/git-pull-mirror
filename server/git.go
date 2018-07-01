package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/url"

	"golang.org/x/crypto/ssh"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// Repository is a git repo that enables to pull and push. Having an instance of this object means that we have a valid repo
type Repository struct {
	repo   *git.Repository
	origin url.GitURL
	target url.GitURL
	client gitClient
}

// Remotes names
const (
	OriginRemote = "origin"
	TargetRemote = "target"
)

func newGitClient(ops WebHooksServerOptions) gitClient {
	return gitClient{ops: ops}
}

type gitClient struct {
	ops WebHooksServerOptions

	repositories []Repository
	wg           *sync.WaitGroup
}

// GitTimeoutSeconds returns the duration in which the context for a git operation will timeout
func (g gitClient) GitTimeoutSeconds() time.Duration {
	return time.Duration(g.ops.GitTimeoutSeconds) * time.Second
}

// CloneOrPull ensures that the repo exists in the indicated path
func (g gitClient) CloneOrOpen(origin url.GitURL, target url.GitURL) (Repository, error) {
	r, err := git.PlainOpen(g.pathFor(origin))
	if err == git.ErrRepositoryNotExists {
		return g.clone(origin, target)
	} else if err != nil {
		return Repository{}, fmt.Errorf("failed to open repo %s: %s", origin, err)
	}

	logrus.Debugf("repository %s already exists locally", origin)
	repo := Repository{
		repo:   r,
		client: g,

		origin: origin,
		target: target,
	}
	return repo, err
}

func (g gitClient) clone(origin url.GitURL, target url.GitURL) (Repository, error) {
	logrus.Debugf("could not find repository %s, cloning into %s", origin, g.pathFor(origin))

	auth, err := g.authMethod(origin)
	if err != nil {
		return Repository{}, fmt.Errorf("failed set up auth to clone origin %s: %s", origin, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.GitTimeoutSeconds())
	defer cancel()

	r, err := git.PlainCloneContext(ctx, g.pathFor(origin), true, &git.CloneOptions{
		URL:          origin.URI,
		Auth:         auth,
		SingleBranch: false,
	})
	if err != nil {
		return Repository{}, fmt.Errorf("failed to execute clone of origin %s: %s", origin, err)
	}

	logrus.Debugf("creating remote `target` for %s", origin)
	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: TargetRemote,
		URLs: []string{target.URI},
	})
	logrus.Debugf("done creating remote `target` for %s", origin)
	if err != nil {
		return Repository{}, fmt.Errorf("failed to add target remote %s to repo %s: %s", target, origin, err)
	}

	return Repository{
		repo:   r,
		client: g,

		origin: origin,
		target: target,
	}, nil
}

func (g gitClient) pathFor(origin url.GitURL) string {
	return filepath.Join(g.ops.RepositoriesPath, origin.ToPath())
}

func (g gitClient) authMethod(uri url.GitURL) (transport.AuthMethod, error) {
	switch uri.Transport {
	case url.GitSSHTransport:
		if g.ops.SSHPrivateKey == "" {
			logrus.Debugf("%s transport for %s but no ssh pk set", uri.Transport, uri)
			break
		}

		logrus.Debugf("loading private key %s for %s", g.ops.SSHPrivateKey, uri)

		pem, err := ioutil.ReadFile(g.ops.SSHPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read ssh private key %s: %s", g.ops.SSHPrivateKey, err)
		}

		signer, err := ssh.ParsePrivateKey(pem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ssh private key %s: %s", g.ops.SSHPrivateKey, err)
		}

		return &gitssh.PublicKeys{
			User:   uri.Username,
			Signer: signer,
		}, nil

	}
	return nil, nil
}

// Fetch pulls from origin
func (r Repository) Fetch() error {
	auth, err := r.client.authMethod(r.origin)
	if err != nil {
		return fmt.Errorf("failed set up auth to fetch from origin %s: %s", r.origin, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.client.GitTimeoutSeconds())
	defer cancel()

	logrus.Debugf("fetching %s", r.origin)
	err = r.repo.FetchContext(ctx, &git.FetchOptions{
		Auth:       auth,
		RemoteName: OriginRemote,
	})
	if err == git.NoErrAlreadyUpToDate {
		logrus.Debugf("%s is already up to date", r.origin)
		return nil
	}
	return err
}

// Push pushes to target
func (r Repository) Push() error {
	auth, err := r.client.authMethod(r.target)
	if err != nil {
		return fmt.Errorf("failed set up auth to push to target %s: %s", r.target, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.client.GitTimeoutSeconds())
	defer cancel()

	logrus.Debugf("pushing to %s", r.target)
	err = r.repo.PushContext(ctx, &git.PushOptions{
		Auth:       auth,
		RemoteName: TargetRemote,
	})
	if err == git.NoErrAlreadyUpToDate {
		logrus.Debugf("%s is already up to date", r.target)
		return nil
	}
	return err
}

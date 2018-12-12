[![pipeline status](https://gitlab.com/yakshaving.art/git-pull-mirror/badges/master/pipeline.svg)](https://gitlab.com/yakshaving.art/git-pull-mirror/commits/master)
[![coverage report](https://gitlab.com/yakshaving.art/git-pull-mirror/badges/master/coverage.svg?job=test)](https://gitlab.com/yakshaving.art/git-pull-mirror/commits/master)

# Git(Lab) Pull Mirror

A simple http server that registers and listens to GitHub Push Webhook events
to then pull and push back to a GitLab instance.

It's a simple way of having pull mirror (an expensive GitLab-EE feature) for
free, with a more responsive implementation.

## Usage

`git-pull-mirror`

This will start the pull mirror webhooks server in the port 9092. It will
load the default configuration file `mirrors.yml` and then will register
itself to GitHub using the $CALLBACK_URL such that webhooks will be
directed to it.

## Environment variables

- **CALLBACK_URL** callback url to report to github for webhooks, must
    include schema and domain.
- **GITHUB_USER** github username, used to configure the webhooks through the
    API.
- **GITHUB_TOKEN** github token, used as the password to configure the
    webhooks through the API
- **SSH_KEY** is the private ssh key used to talk to the remotes. It needs to
    be explicitly set, there will be no assumptions made around which ssh key to
    use.

## Options

- **-callback.url** *string*
    callback url to report to github for webhooks, must include schema and domain (default loaded from env CALLBACK_URL)
- **-config.file** *string*
    configuration file (default "mirrors.yml")
- **-debug**
    enable debugging log level
- **-pprof.address**
    address in which to listen for pprof debugging requests
- **-dryrun**
    execute configuration loading then exit. Don't actually do anything
- **-git.timeout.seconds** *int*
    git operations timeout in seconds, defaults to 60 (default 60)
- **-github.token** *string*
    github token, used as the password to configure the webhooks through the API (default loaded from env GITHUB_TOKEN)
- **-github.url** *string*
    api url to register webhooks (default "https://api.github.com/hub")
- **-github.user** *string*
    github username, used to configure the webhooks through the API (default loaded from env GITHUB_USER)
- **-listen.address** *string*
    address in which to listen for webhooks (default ":9092")
- **-repositories.path** *string*
    local path in which to store cloned repositories (default ".")
- **-sshkey** *string*
    ssh key to use to identify to remotes

## Signals

**git-pull-mirrors** supports at least 4 signals:

- **SIGINT** will perform a graceful shutdown in which it will stop accepting
    webhooks, then finish all the pending work to then exit.
- **SIGHUP** will reload the mirrors.yml configuration file and apply it
    without downtime. If configuration parsing fails, it will not be applied.
- **SIGUSR1** will toggle log debugging on and off.
- **SIGUSR2** will trigger a full update process for all the registered mirrors

## Metrics

**git-pull-mirrors** offers prometheus metrics used to track the state of the service, these should be used to monitor that the service is operating correctly.

| name | type | help  |
|---|---|---|
| github_webhooks_up                            | gauge    | whether the service is ready to receive requests or not |
| github_webhooks_repo_up                       | gauge    | whether a repo is succeeding or failing to read or write |
| github_webhooks_git_latency_seconds           | summary  | latency percentiles of git fetch and push operations |
| github_webhooks_hooks_received_total          | counter  | total count of hooks received |
| github_webhooks_hooks_retried_total           | counter  | total number of hooks that failed and were retried |
| github_webhooks_hooks_updated_total           | counter  | total number of repos succefully updated  |
| github_webhooks_hooks_failed_total            | counter  | total number of repos that failed to update for some reason  |
| github_webhooks_boot_time_seconds             | gauge    | unix timestamp indicating when the process was started |
| github_webhooks_last_successful_config_apply  | gauge    | unix timestamp indicating when the last configuration reload was successfully executed  |

## Running

### Cloud native way
TODO. C'mon, that's probably one (albeit long) line starting with `docker run` or `kubectl`.

### Oldschool way
Here are snippets from the instance startup shell script that will install, enable, configure and run
the git-pull-mirror as a systemd service in an idempotent way. The exercises of combining them together,
managing the secrets, and configuring `gitlab.rb` so that nginx serves `/hooks` (hint: no SSL in git-pull-mirror)
and `/metrics` endpoints are left to the reader (hint: `nginx['custom_gitlab_server_config']` should do).

Installing:
```bash
PULL_MIRROR_VERSION=0.0.6
PULL_MIRROR_BINARY='/usr/local/bin/git-pull-mirror'

if [[ ! -f "$PULL_MIRROR_BINARY" ]]; then
	wget -q -P /tmp "https://github.com/yakshaving-art/git-pull-mirror/releases/download/$PULL_MIRROR_VERSION/git-pull-mirror_${PULL_MIRROR_VERSION}_linux_amd64.tar.gz"
	wget -qO - "https://github.com/yakshaving-art/git-pull-mirror/releases/download/$PULL_MIRROR_VERSION/git-pull-mirror_${PULL_MIRROR_VERSION}_checksums.txt" | \
		sed -n '/linux_amd64/ s|  |  /tmp/|p;' | \
		sha256sum --quiet --check # will crap out on mismatch
	tar zxf "/tmp/git-pull-mirror_${PULL_MIRROR_VERSION}_linux_amd64.tar.gz" -C "$(dirname "$PULL_MIRROR_BINARY")"
	chmod 0755 "$PULL_MIRROR_BINARY"
	rm -f "/tmp/git-pull-mirror_${PULL_MIRROR_VERSION}_linux_amd64.tar.gz"
fi
```

Adding user and its ssh keys:
```bash
PULL_MIRROR_USER="techguru"
PULL_MIRROR_HOME="/home/$PULL_MIRROR_USER"
getent passwd "$PULL_MIRROR_USER" || \
	adduser --home "$PULL_MIRROR_HOME" \
		--disabled-login \
		--disabled-password \
		--gecos "aka The Mirrorbot" \
		"$PULL_MIRROR_USER"

# copy .ssh from secret storage, or regen with:
# ssh-keygen -t ed25519 -C 'MirrorBot key: techguru@mygtlb' -N '' -f id_ed25519_techguru
cp -rp "/path/to/secrets/folder/mirrorbot-.ssh" "$PULL_MIRROR_HOME/.ssh"
chown -R "$PULL_MIRROR_USER:$PULL_MIRROR_USER" "$PULL_MIRROR_HOME"
chmod 0700 "$PULL_MIRROR_HOME/.ssh"
chmod 0400 "$PULL_MIRROR_HOME/.ssh/id_ed25519_techguru"
```

Set up env file for service (add fingerprints to `known_hosts` file, or just treat it
as secret and copy from secret storage during boot):
``` bash
test -f /etc/gitlab/git-pull-mirror.conf || cat > /etc/gitlab/git-pull-mirror.conf <<EOF
SSH_KEY=".ssh/id_ed25519_techguru"
SSH_KNOWN_HOSTS=".ssh/known_hosts"
CALLBACK_URL="https://gitlab.my.tld/hooks"
GITHUB_USER="my-cbot"
GITHUB_TOKEN="pszsetme"
EOF
```

Create systemd unit file (sysV init script left as an exercise
for even older school):

```bash
cat > /etc/systemd/system/git-pull-mirror.service <<EOF
[Unit]
Description=Git Pull Mirror
After=gitlab-runner.target
ConditionFileIsExecutable=$PULL_MIRROR_BINARY
[Service]
User=$PULL_MIRROR_USER
WorkingDirectory=$PULL_MIRROR_HOME
EnvironmentFile=/etc/gitlab/git-pull-mirror.conf
StartLimitInterval=5
StartLimitBurst=10
ExecStart=$PULL_MIRROR_BINARY -listen.address="127.0.0.1:9092" -config.file="$PULL_MIRROR_HOME/mirrors.yml" -skip.webhooks.registration="true" -debug
Restart=always
RestartSec=120
[Install]
WantedBy=multi-user.target
EOF
chmod 0644 /etc/systemd/system/git-pull-mirror.service
```

Create `mirrors.yml` in the proper location:

```bash
cat > "$PULL_MIRROR_HOME/mirrors.yml" <<EOF
---
repositories:
- origin: git@github.com:source/source-repo1.git
  target: git@gitlab.my.tld:dst/dst-repo1.git

- origin: git@github.com:source/source-repo2.git
  target: git@gitlab.my.tld:dst/dst-repo2.git
EOF
chown "$PULL_MIRROR_USER:$PULL_MIRROR_USER" "$PULL_MIRROR_HOME/mirrors.yml"
chmod 0644 "$PULL_MIRROR_HOME/mirrors.yml"
```

Don't forget to add pubkey to both github and gitlab, and do a:
```bash
systemctl daemon-reload
systemctl enable git-pull-mirror.service
systemctl start git-pull-mirror.service
```

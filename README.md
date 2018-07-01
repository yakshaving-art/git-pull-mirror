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

- -callback.url string
    callback url to report to github for webhooks, must include schema and domain (default loaded from env CALLBACK_URL)
- -config.file string
    configuration file (default "mirrors.yml")
- -debug
    enable debugging log level
- -dryrun
    execute configuration loading then exit. Don't actually do anything
- -git.timeout.seconds int
    git operations timeout in seconds, defaults to 60 (default 60)
- -github.token string
    github token, used as the password to configure the webhooks through the API (default loaded from env GITHUB_TOKEN)
- -github.url string
    api url to register webhooks (default "https://api.github.com/hub")
- -github.user string
    github username, used to configure the webhooks through the API (default loaded from env GITHUB_USER)
- -listen.address string
    address in which to listen for webhooks (default "localhost:9092")
- -repositories.path string
    local path in which to store cloned repositories (default ".")
- -sshkey string
    ssh key to use to identify to remotes

## Signals

**git-pull-mirrors** supports at least 3 signals:

- **SIGINT** will perform a graceful shutdown in which it will stop accepting
    webhooks, then finish all the pending work to then exit.
- **SIGHUP** will reload the mirrors.yml configuration file and apply it
    without downtime. If configuration parsing fails, it will not be applied.
- **SIGUSR1** will toggle log debugging on and off.

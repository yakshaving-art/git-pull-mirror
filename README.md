[![pipeline status](https://gitlab.com/yakshaving.art/git-pull-mirror/badges/master/pipeline.svg)](https://gitlab.com/yakshaving.art/git-pull-mirror/commits/master)
[![coverage report](https://gitlab.com/yakshaving.art/git-pull-mirror/badges/master/coverage.svg?job=test)](https://gitlab.com/yakshaving.art/git-pull-mirror/commits/master)

# Git(Lab) Pull Mirror

A simple http server that registers and listens to GitHub Push Webhook events
to then pull and push back to a GitLab instance.

It's a simple way of having pull mirror (an expensive GitLab-EE feature) for
free, with a more responsive implementation.

## Usage

`git-pull-mirror public-hostname`

This will start the pull mirror webhooks server in the port 9092. It will
load the default configuration file `mirrors.yml` and then will register
itself to GitHub using the public-hostname such that webhooks will be
directed to `https://public-hostname/hooks/:owner/:name`

## Environment variables

* **GITHUB_USERNAME** is the username that will register the webhooks
* **GITHUB_TOKEN** is the token used as a password to register the webhooks

## Options

```
-config.file string
    configuration file (default "mirrors.yml")
-debug
    enable debugging log level
-dryrun
    execute configuration loading then exit. Don't actually do anything
-git.timeout.seconds int
    git operations timeout in seconds, defaults to 60 (default 60)
-listen.address string
    address in which to listen for webhooks (default "localhost:9092")
-repositories.path string
    local path in which to store cloned repositories (default ".")
```

## Signals

`git-pull-mirrors` supports at least 3 signals:

* **SIGINT** will perform a graceful shutdown in which it will stop accepting
  webhooks, then finish all the pending work to then exit.
* **SIGHUP** will reload the mirrors.yml configuration file and apply it
  without downtime. If configuration parsing fails, it will not be applied.
* **SIGUSR1** will toggle log debugging on and off.

## Bump

A generic version tracking and update tool.

Bump can be used to automate version updates where other version and package
management system does not fit or can't be used. This can be for example when
having versions of dependencies in Makefiles, Dockerfiles, scripts or other
kinds of texts.

For example this is a Dockerfile where we want to keep the base image version
updated to the latest exact alpine 3 version.

```sh (exec)
$ cat examples/Dockerfile

# See possible updates
$ bump check examples/Dockerfile

# See what will be changed
$ bump diff examples/Dockerfile

# Write changes and run commands
$ bump update examples/Dockerfile
```

A real world example is the
[Dockerfile used by wader/static-ffmpeg](https://github.com/wader/static-ffmpeg/blob/master/Dockerfile)
where important libraries are automatically kept up to date using the bump github action.

## GitHub action

Bump can be used as a github action using the action `wader/bump@master`
or by [providing it and referencing yourself](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/configuring-a-workflow#referencing-actions-in-your-workflow)
Here is a workflow that will read `Bumpfile` and look for new versions and creates PRs once per day at 9 UTC:

```yml
name: 'Automatic version updates'
on:
  schedule:
    # minute hour dom month dow (UTC)
    - cron: '0 9 * * *'
jobs:
  bump:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@master
      - uses: wader/bump@master
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

See [action.yml](action.yml) for input arguments.

Note that if you want bump PRs to trigger other actions like CI builds
[you currently have to use a personal access token](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/events-that-trigger-workflows#about-workflow-events)
with repo access and add it as a secret. For example
add a secret named `BUMP_TOKEN` and do `GITHUB_TOKEN: ${{ secrets.BUMP_TOKEN }}`.

See [Dockerfile](Dockerfile) for tools installed in the default image.

## Install

### Docker

Image from docker hub:
```sh
docker run --rm -v "$PWD:$PWD" -w "$PWD" mwader/bump help
```
Build image:
```sh
docker build -t bump .
```

### Build

Install go and run command below and it will be installed at
`$(go env GOPATH)/bin/bump`.

```sh
go get github.com/wader/bump/cmd/bump
```

## Usage

```sh (exec)
$ bump help
```

## Configuration

`NAME` is a name of the configuration.

`REGEXP` is a [golang regexp](https://golang.org/pkg/regexp/syntax/) with
one submatch/capture group to find the current version.

`PIPELINE` is a pipeline of filters that describes how to find the latest
suitable version. The syntax is similar to pipes in a shell `filter|filter|...`
where `filter` is either in the form `name:argument` like `re:/[\d.]+/`,
`semver:^4` or a shorter form like `/[\d.]+/`, `^4` etc.

### Bumpfile

Default `bump` looks for a file named `Bumpfile` in the current directory.
Each line is a comment, configuration or a glob pattern of files to
read embedded configuration from.

```
# comment
NAME /REGEXP/ PIPELINE
NAME [command|after] COMMAND
NAME message MESSAGE
NAME link "TITLE" URL
filename
glob/*
```

Example Bumpfile:

```
# a bump configuration
alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
alpine message Make sure to also test with abc
alpine link "Release notes" https://alpinelinux.org/posts/Alpine-$LATEST-released.html
# read configuration, check and update version in Dockerfile
Dockerfile
```

### Embedded

Embedded configuration can be used to include bump configuration inside
files containing versions to be checked or updated.

Embedded configuration looks like this:
```
bump: NAME /REGEXP/ PIPELINE
```

Example Dockerfile with embedded configuration:
```
# bump: alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
FROM alpine:3.9.3 AS builder
```

### Run shell command on update

```
bump: NAME [command|after] COMMAND
```

There are two kinds of shell commands, `command` and `after`. `command` will be executed
instead bump doing the changes. `after` will always be executed after bump has done any changes.
If you have multiple commands they will be executed in the same order as they are configured.

Example Bumpfile using `command` to run `go get` to change `go.mod` and `go.sum`:
```
module program

go 1.12

require (
  // bump: leaktest /github.com\/fortytw2\/leaktest v(.*)/ git:https://github.com/fortytw2/leaktest.git|^1
  // bump: leaktest command go get github.com/fortytw2/leaktest@v$LATEST && go mod tidy
  github.com/fortytw2/leaktest v1.2.0
)
```

Example Bumpfile using `after` to run a script to update download hashes:
```
libvorbis after ./hashupdate Dockerfile VORBIS $LATEST
```

### Commit and pull request messages and links

```
NAME message MESSAGE
NAME link "TITLE" URL
```

You can include messages and links in commit messages and pull requests by using one or
more `message` and `link` configurations.

Example:
```
libvorbis link "CHANGES file" https://github.com/xiph/vorbis/blob/master/CHANGES
libvorbis link "Source diff $CURRENT..$LATEST" https://github.com/xiph/vorbis/compare/v$CURRENT..v$LATEST
```


## Pipeline

A pipeline consist of one or more filters executed in sequence. Usually
it starts with a filter that produces versions from some source like a git repository.
After that one or more filters can select, transform and sort versions to narrow it
down to one version. If a pipeline ends up producing more than one version the first
will be used.

A version is a dictionary of key/value pairs, the "name" key is either the version number
like "1.2.3" or some symbolic name like "master". In addition a version can have other keys
like "commit", "version" etc depending on the source. You can use the key filter `key:<name>`
or `@<name>` to use them.

Default all filters operate on the default key which is the "name". This can be changed
along a pipeline using `key:<name>` or `@<name>`.

### Examples

In the examples `bump pipeline PIPELINE` is used to test run a pipeline and show
the result. Use `bump -v pipeline PIPELINE` for even more verbose output that
can be helpful when testing pipelines.

```sh (exec)
# Latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4'

# Commit hash of the latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4|@commit'

# Latest 1.0 golang docker build image
$ bump pipeline 'docker:golang|^1'

# Latest mp3lame version
$ bump pipeline 'svn:http://svn.code.sf.net/p/lame/svn|/^RELEASE__(.*)$/|/_/./|*'
```

## Filters

Filter are used to produce, transform and filter versions. Some filters like `git`
produces versions, `re` and `semver` transforms and filters.

[filtersmarkdown]: sh

## Ideas, TODOs and known issues

- GitHub action: How to access package manager tools? separate docker images? bump-go?
- GitHub action: PR labels
- GitHub action: some kind of tests
- Configuration templates, go package etc?
- Proper version number for bump itself
- How to use with hg
- docker filter: value should be layer hash
- docker filter: support auth and other registries
- Named pipelines, "ffmpeg|^4", generate URLs to changelog/diff?
- Allow to escape `|` in filter argument
- Allow alternative regexp "/" delimiter in version match or maybe some simplified match syntax?
- sort filter: make smarter? natural sort?
- Some kind of cache to better handle multiple invocations
- HTTP service to run pipelines?

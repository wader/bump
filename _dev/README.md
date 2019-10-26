## Bump

A generic version tracking tool.

Bump can be used to automate version updates where other version and package
management system does not fit or can't be used. This can be for example when
having versions of dependencies in Makefiles, Dockerfiles, scripts or other
kinds of texts.

For example this is a Dockerfile where we want to keep the base image version
updated to the latest exact alpine 3.0 version.

```sh (exec)
$ cat examples/Dockerfile

# See possible updates
$ bump check examples/Dockerfile

# See what will be changed
$ bump diff examples/Dockerfile

# Write changes
$ bump update examples/Dockerfile
```

## GitHub action

Bump can be used as a github action using the action `wader/bump@master`.
For example this workflow will look for new versions and creates PRs
one time per day.

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
        with:
          bump_files: file1 file2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

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

`bump` looks for lines looking like this:
```
bump: NAME /REGEXP/ PIPELINE
```

`NAME` is a name of the software etc this configuration is for.

`REGEXP` is a [golang regexp](https://golang.org/pkg/regexp/syntax/) with
one submatch/capture group to find the current version.

`PIPELINE` is a pipeline of filters that describes how to find the latest
suitable version. The syntax is similar to pipes in a shell `filter|filter|...`
where `filter` is either in the form `name:argument` like `re:/[\d.]+/`,
`semver:^4` or a shorter form like `/[\d.]+/`, `^4` etc.

Usually the lines will be in comments or in a separate file.

## Pipeline

A pipeline consist of one or more filters executed in sequence. Usually
starts with a filter that produces versions from a source like a git repository.
After that usually zero or more filters are used to narrow down to one version.
The first version will be used if the last filter produces more than one.

A version can optionally have a associated value that can be for example a hash
of the git tag used as version. Use the `value`/`@` filter last in a pipeline to
use it.

### Examples

```sh (exec)
# Latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4'

# Commit hash of the latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4|@'

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

- GitHub action: some kind of tests
- Proper version number for bump itself
- Some kind of Bumpfile with config and paths to check for updates?
- How to use with hg
- docker filter: value should be layer hash
- docker filter: support auth and other registries
- Named pipelines, "ffmpeg|^4", generate URLs to changelog/diff?
- Allow to escape `|` in filter argument
- sort filter: make smarter? natural sort?
- Some kind of cache to better handle multiple invocations
- HTTP service to run pipelines?

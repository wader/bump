## Bump

A generic version tracking and update tool.

Bump can be used to automate version updates where other version and package
management system does not fit or can't be used. This can be for example when
having versions of dependencies in Makefile:s, Dockerfile:s, scripts or other
kinds of texts.

For example this is a Bumpfile where we want to keep the Dockerfile base image
version updated to the latest exact alpine 3 version.

```sh (exec)
$ cat Bumpfile
# Configuration for "alpine"
# <name> <regexp to match version> <pipeline>
alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
# Links to include in commit
alpine link "Release notes" https://alpinelinux.org/posts/Alpine-$LATEST-released.html
# Look for matches in Dockerfile
Dockerfile
# See current versions
$ bump current
Dockerfile:1: alpine 3.9.2
# See possible updates
$ bump check
alpine 3.23.0
# See what will be changed
$ bump diff
--- Dockerfile
+++ Dockerfile
@@ -1,2 +1,2 @@
-FROM alpine:3.9.2 AS builder
+FROM alpine:3.23.0 AS builder
 
# Write changes
$ bump update
```

It's also possible to have configuration embedded in source code comments etc and it's also possible to specify files to check instead of using a `Bumpfile`.

A real world example is the
[Dockerfile used by wader/static-ffmpeg](https://github.com/wader/static-ffmpeg/blob/master/Dockerfile)
where important libraries are automatically kept up to date using the bump github action.

## GitHub action

Bump can be used as a github action using the action `wader/bump/action@master`
or by [providing it and referencing yourself](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/configuring-a-workflow#referencing-actions-in-your-workflow)
Here is a workflow that will read `Bumpfile` and look for new versions and creates PRs once per day at 9 UTC:

```yml
name: 'Automatic version updates'
on:
  schedule:
    # minute hour dom month dow (UTC)
    - cron: '0 9 * * *'
  # enable manual trigger of version updates
  workflow_dispatch:
jobs:
  bump:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: wader/bump/action@master
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

See [action.yml](action/action.yml) for input arguments.

Note that if you want bump PRs to trigger other actions like CI builds
[you currently have to use a personal access token](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/events-that-trigger-workflows#about-workflow-events)
with repo access and add it as a secret. For example
add a secret named `BUMP_TOKEN` and do `GITHUB_TOKEN: ${{ secrets.BUMP_TOKEN }}`.

These actions with different environments are available:  
`wader/bump/action@master` alpine with git and curl  
`wader/bump/action/go@master` alpine with git, curl and go  

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
Usage: bump [OPTIONS] COMMAND
OPTIONS:
  -e                    Comma separated names to exclude
  -f                    Bumpfile to read (Bumpfile)
  -i                    Comma separated names to include
  -r                    Run update commands (false)
  -v                    Verbose (false)

COMMANDS:
  version               Show version of bump itself (dev)
  help [FILTER]         Show help or help for a filter
  list [FILE...]        Show bump configurations
  current [FILE...]     Show current versions
  check [FILE...]       Check for possible version updates
  update [FILE...]      Update versions
  diff [FILE...]        Show diff of what an update would change
  pipeline PIPELINE     Run a filter pipeline

EXIT CODE:
  0: All went fine
  1: Something went wrong
  3: Check found new versions

BUMPFILE is a file with CONFIG:s or glob patterns of FILE:s
FILE is a file with EMBEDCONFIG:s or versions to be checked and updated
EMBEDCONFIG is "bump: CONFIG"
CONFIG is
  NAME /REGEXP/ PIPELINE |
  NAME command COMMAND |
  NAME after COMMAND |
  NAME message MESSAGE |
  NAME link TITLE URL
NAME is a configuration name
REGEXP is a regexp with one submatch to find current version
PIPELINE is a filter pipeline: FILTER|FILTER|...
FILTER
  git:<repo> | <repo.git>
  gitrefs:<repo>
  depsdev:<system>:<package>
  docker:<image>
  svn:<repo>
  fetch:<url> | <http://> | <https://>
  semver:<constraint> | semver:<n.n.n-pre+build> | <constraint> | <n.n.n-pre+build>
  re:/<regexp>/ | re:/<regexp>/<template>/ | /<regexp>/ | /<regexp>/<template>/
  sort
  key:<name> | @<name>
  static:<name[:key=value:...]>,...
  err:<error>
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
NAME link TITLE URL
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
`COMMAND` will run with these environment variables set:  
`$NAME` is configuration name  
`$CURRENT` is current version  
`$LATEST` is latest version available  

There are two kinds of shell commands, `command` and `after`. `command` will be executed
instead bump doing the changes. `after` will always be executed after bump has done any changes.
If you have multiple commands they will be executed in the same order as they are configured.

Example Bumpfile using `command` to run `go get` to change `go.mod` and `go.sum`:
```
module program

go 1.12

require (
  // bump: leaktest /github.com\/fortytw2\/leaktest v(.*)/ git:https://github.com/fortytw2/leaktest.git|^1
  // bump: leaktest command go get -d github.com/fortytw2/leaktest@v$LATEST && go mod tidy
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
These variable are available in `MESSAGE`, `TITLE` and `URL`:  
`$NAME` is configuration name  
`$CURRENT` is current version  
`$LATEST` is latest version available  

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
4.4.6
# Commit hash of the latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4|@commit'
784eb97e010cee6bb4d81c352bd6eed4b7dedda2
# Latest 1.0 golang docker build image
$ bump pipeline 'docker:golang|^1'
1.25.5
# Latest mp3lame version
$ bump pipeline 'svn:http://svn.code.sf.net/p/lame/svn|/^RELEASE__(.*)$/|/_/./|*'
3.100
```

## Filters

Filter are used to produce, transform and filter versions. Some filters like `git`
produces versions, `re` and `semver` transforms and filters.

[filtersmarkdown]: sh-start

[git](#filter-git) `git:<repo>` or `<repo.git>`<br>
[gitrefs](#filter-gitrefs) `gitrefs:<repo>`<br>
[depsdev](#filter-depsdev) `depsdev:<system>:<package>`<br>
[docker](#filter-docker) `docker:<image>`<br>
[svn](#filter-svn) `svn:<repo>`<br>
[fetch](#filter-fetch) `fetch:<url>`, `<http://>` or `<https://>`<br>
[semver](#filter-semver) `semver:<constraint>`, `semver:<n.n.n-pre+build>`, `<constraint>` or `<n.n.n-pre+build>`<br>
[re](#filter-re) `re:/<regexp>/`, `re:/<regexp>/<template>/`, `/<regexp>/` or `/<regexp>/<template>/`<br>
[sort](#filter-sort) `sort`<br>
[key](#filter-key) `key:<name>` or `@<name>`<br>
[static](#filter-static) `static:<name[:key=value:...]>,...`<br>
[err](#filter-err) `err:<error>`<br>
### git<span id="filter-git">

`git:<repo>` or `<repo.git>`

Produce versions from tags for a git repository. Name will be
the version found in the tag, commit the commit hash or tag object.

Use gitrefs filter to get all refs unfiltered.

```sh
$ bump pipeline 'https://github.com/git/git.git|*'
2.52.0
```

### gitrefs<span id="filter-gitrefs">

`gitrefs:<repo>`

Produce versions from all refs for a git repository. Name will be the whole ref
like "refs/tags/v2.7.3" and commit will be the commit hash.

Use git filter to get versions from only tags.

```sh
$ bump pipeline 'gitrefs:https://github.com/git/git.git'
HEAD
```

### depsdev<span id="filter-depsdev">

`depsdev:<system>:<package>`

Produce versions from https://deps.dev.

Supported package systems npm, go, maven, pypi and cargo.

```sh
$ bump pipeline 'depsdev:npm:react|*'
19.2.1
$ bump pipeline 'depsdev:go:golang.org/x/net'
0.0.0-20120125194513-f61fbb80d2fc
$ bump pipeline 'depsdev:maven:log4j:log4j|^1'
1.2.17
$ bump pipeline 'depsdev:pypi:av|*'
16.0.1
$ bump pipeline 'depsdev:cargo:serde|*'
1.0.228
```

### docker<span id="filter-docker">

`docker:<image>`

Produce versions from a image on docker hub or other registry.
Currently only supports anonymous access.

```sh
$ bump pipeline 'docker:alpine|^3'
3.23.0
$ bump pipeline 'docker:mwader/static-ffmpeg|^4'
4.4.1
$ bump pipeline 'docker:ghcr.io/nginx-proxy/nginx-proxy|^0.9'
0.9.3
```

### svn<span id="filter-svn">

`svn:<repo>`

Produce versions from tags and branches from a subversion repository. Name will
be the tag or branch name, version the revision.

```sh
$ bump pipeline 'svn:https://svn.apache.org/repos/asf/subversion|*'
1.14.5
```

### fetch<span id="filter-fetch">

`fetch:<url>`, `<http://>` or `<https://>`

Fetch a URL and produce one version with the content as the key "name".

```sh
$ bump pipeline 'fetch:http://libjpeg.sourceforge.net|/latest release is version (\w+)/'
6b
```

### semver<span id="filter-semver">

`semver:<constraint>`, `semver:<n.n.n-pre+build>`, `<constraint>` or `<n.n.n-pre+build>`

Use [semver](https://semver.org/) to filter or transform versions.

When a constraint is provided it will be used to find the latest version fulfilling
the constraint.

When a version pattern is provided it will be used to transform a version.

```sh
# find latest major 1 version
$ bump pipeline 'static:1.1.2,1.1.3,1.2.0|semver:^1'
1.2.0
# find latest minor 1.1 version
$ bump pipeline 'static:1.1.2,1.1.3,1.2.0|~1.1'
1.1.3
# transform into just major.minor
$ bump pipeline 'static:1.2.3|n.n'
1.2
```

### re<span id="filter-re">

`re:/<regexp>/`, `re:/<regexp>/<template>/`, `/<regexp>/` or `/<regexp>/<template>/`

An alternative regex/template delimited can specified by changing the first
/ into some other character, for example: re:#regexp#template#.

Filter name using a [golang regexp](https://golang.org/pkg/regexp/syntax/).
If name does not match regexp the version will be skipped.

If only a regexp and no template is provided and no submatches are defined the
name will not be changed.

If submatches are defined a submatch named "name" or "value" will be used as
name and value otherwise first submatch will be used as name.

If a template is defined and no submatches was defined it will be used as a
replacement string. If submatches are defined it will be used as a template
to expand $0, ${1}, $name etc.

A regexp can match many times. Use ^$ anchors or (?m:) to match just one time
or per line.

```sh
# just filter
$ bump pipeline 'static:a,b|/b/'
b
# simple replace
$ bump pipeline 'static:aaa|re:/a/b/'
bbb
# simple replace with # as delimiter
$ bump pipeline 'static:aaa|re:#a#b#'
bbb
# name as first submatch
$ bump pipeline 'static:ab|re:/a(.)/'
b
# multiple submatch replace
$ bump pipeline 'static:ab:1|/(.)(.)/${0}$2$1/'
abba
# named submatch as name and value
$ bump pipeline 'static:ab|re:/(?P<name>.)(?P<value>.)/'
a
$ bump pipeline 'static:ab|re:/(?P<name>.)(?P<value>.)/|@value'
b
```

### sort<span id="filter-sort">

`sort`

Sort versions reverse alphabetically.

```sh
$ bump pipeline 'static:a,b,c|sort'
c
```

### key<span id="filter-key">

`key:<name>` or `@<name>`

Change default key for a pipeline. Useful to have last in a pipeline
to use git commit hash instead of tag name etc or in the middle of
a pipeline if you want to regexp filter on something else than name.

```sh
$ bump pipeline 'static:1.0:hello=world|@hello'
world
$ bump pipeline 'static:1.0:hello=world|@name'
1.0
$ bump pipeline 'static:1.0:hello=world|key:hello'
world
```

### static<span id="filter-static">

`static:<name[:key=value:...]>,...`

Produce versions from filter argument.

```sh
$ bump pipeline 'static:1,2,3,4:key=value:a=b|sort'
4
```

### err<span id="filter-err">

`err:<error>`

Fail with error message. Used for testing.

```sh
$ bump pipeline 'err:test'
test
```


[#]: sh-end

## Ideas, TODOs and known issues

- GitHub action: PR labels
- GitHub action: some kind of tests
- Configuration templates, go package etc?
- Proper version number for bump itself
- How to use with hg
- docker filter: value should be layer hash
- docker filter: support non-anon-auth
- Named pipelines, "ffmpeg|^4", generate URLs to changelog/diff?
- Allow to escape `|` in filter argument
- Sort filter: make smarter? natural sort?
- Custom verison sort filter somehow, similar to `sort -k` etc?
- Some kind of cache to better handle multiple invocations
- HTTP service to run pipelines?
- bump-ng: Use jq or some other pipe-friednly langauge
- Some kind help to build URLs that have major.mainor etc, ex: https://host/name-1.2/name-1.3.4.tar.gz

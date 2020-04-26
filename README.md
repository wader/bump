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
# bump: alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
FROM alpine:3.9.3 AS builder

# See possible updates
$ bump check examples/Dockerfile
alpine 3.11.6

# See what will be changed
$ bump diff examples/Dockerfile
--- examples/Dockerfile
+++ examples/Dockerfile
@@ -1,2 +1,2 @@
 # bump: alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
-FROM alpine:3.9.3 AS builder
+FROM alpine:3.11.6 AS builder

# Write changes
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
  -c                    Bumpfile to read (Bumpfile)
  -e                    Comma separated names to exclude
  -i                    Comma separated names to include
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

BUMPFILE is a file with CONFIG:s or glob patterns of FILE:s
FILE is file with EMBEDCONFIG:s or versions to be checked or updated
CONFIG is "NAME /REGEXP/ PIPELINE"
EMBEDCONFIG is "bump: CONFIG"
PIPELINE is a filter pipeline: FILTER|FILTER|...
FILTER
  git:<repo> | <repo.git>
  gitrefs:<repo>
  docker:<image>
  svn:<repo>
  fetch:<url> | <http://> | <https://>
  semver:<constraint> | semver:<n.n.n-pre+build> | <constraint> | <n.n.n-pre+build>
  re:/<regexp>/ | re:/<regexp>/<template>/ | /<regexp>/ | /<regexp>/<template>/
  sort
  value | @
  static:<name[:value]>,...
  err:<error>
NAME is a configuration name
REGEXP is a regexp with one submatch to find current version
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
glob/*
```

Example Bumpfile:

```
# a bump configuration
alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
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


## Pipeline

A pipeline consist of one or more filters executed in sequence. Usually
it starts with a filter that produces versions from a source like a git repository.
After that one or filters can be used to narrow down to one version.
If a pipeline ends up producing more than one version the first will be used.

A version can optionally have a associated value that can for example in the git case
be the commit hash of the tag. To use the value instead of the version use the
`value` or `@` filter last in a pipeline.

### Examples

In the examples `bump pipeline PIPELINE` is used to test run a pipeline and show
the result. Use `bump -v pipeline PIPELINE` for even more verbose output that
can be helpful when testing pipelines.

```sh (exec)
# Latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4'
4.2.2

# Commit hash of the latest 4.0 ffmpeg version
$ bump pipeline 'https://github.com/FFmpeg/FFmpeg.git|^4|@'
b53940e13dde81d721621b4d5296eede5795aadd

# Latest 1.0 golang docker build image
$ bump pipeline 'docker:golang|^1'
1.14.2

# Latest mp3lame version
$ bump pipeline 'svn:http://svn.code.sf.net/p/lame/svn|/^RELEASE__(.*)$/|/_/./|*'
3.100
```

## Filters

Filter are used to produce, transform and filter versions. Some filters like `git`
produces versions, `re` and `semver` transforms and filters.

[filtersmarkdown]: sh
[git](#git) `git:<repo>` or `<repo.git>`  
[gitrefs](#gitrefs) `gitrefs:<repo>`  
[docker](#docker) `docker:<image>`  
[svn](#svn) `svn:<repo>`  
[fetch](#fetch) `fetch:<url>`, `<http://>` or `<https://>`  
[semver](#semver) `semver:<constraint>`, `semver:<n.n.n-pre+build>`, `<constraint>` or `<n.n.n-pre+build>`  
[re](#re) `re:/<regexp>/`, `re:/<regexp>/<template>/`, `/<regexp>/` or `/<regexp>/<template>/`  
[sort](#sort) `sort`  
[value](#value) `value` or `@`  
[static](#static) `static:<name[:value]>,...`  
[err](#err) `err:<error>`  
### git

`git:<repo>` or `<repo.git>`

Produce versions from tags for a git repository. Name will be
the version found in the tag, value the commit hash or tag object.

Use gitrefs filter to get all refs unfiltered.

```sh
$ bump pipeline 'https://github.com/git/git.git|*'
2.26.2
$ bump pipeline 'git://github.com/git/git.git|*'
2.26.2
```

### gitrefs

`gitrefs:<repo>`

Produce versions from all refs for a git repository.

Use git filter to get versions from only tags.

```sh
$ bump pipeline 'gitrefs:https://github.com/git/git.git'
HEAD
```

### docker

`docker:<image>`

Produce versions from a image on ducker hub.

```sh
$ bump pipeline 'docker:alpine|^3'
3.11.6
```

### svn

`svn:<repo>`

Produce versions from tags and branches from a subversion repository. Name will
be the tag or branch name, value the revision.

```sh
$ bump pipeline 'svn:https://svn.apache.org/repos/asf/subversion|*'
1.13.0
```

### fetch

`fetch:<url>`, `<http://>` or `<https://>`

Fetch a URL and produce one version pair with the content as name.

```sh
$ bump pipeline 'fetch:http://libjpeg.sourceforge.net|/latest release is version (\w+)/'
6b
```

### semver

`semver:<constraint>`, `semver:<n.n.n-pre+build>`, `<constraint>` or `<n.n.n-pre+build>`

Use [semver](https://semver.org/) to filter or transform versions.

When a constraint is provided it will be used to find the latest version fulfilling
the constraint.

When a verison pattern is provied it will be used to transform a version.

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

### re

`re:/<regexp>/`, `re:/<regexp>/<template>/`, `/<regexp>/` or `/<regexp>/<template>/`

An alternative regex/template delimited can specified by changing the first
/ into some other character, for example: re:#regexp#template#.

Filter name using a [golang regexp](https://golang.org/pkg/regexp/syntax/).
If name does not match regexp the pair will be skipped.

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
$ bump pipeline 'static:ab|re:/(?P<name>.)(?P<value>.)/|@'
b
```

### sort

`sort`

Sort versions reverse alphabetically.

```sh
$ bump pipeline 'static:a,b,c|sort'
c
```

### value

`value` or `@`

Use value instead of name.

```sh
$ bump pipeline 'static:a:1|@'
1
$ bump pipeline 'static:a:1|value'
1
```

### static

`static:<name[:value]>,...`

Produce version pairs from filter argument.

```sh
$ bump pipeline 'static:1,2,3|sort'
3
```

### err

`err:<error>`

Fail with error message. Used for testing.

```sh
$ bump pipeline 'err:test'
test
```


## Ideas, TODOs and known issues

- GitHub action: some kind of tests
- Proper version number for bump itself
- How to use with hg
- docker filter: value should be layer hash
- docker filter: support auth and other registries
- Named pipelines, "ffmpeg|^4", generate URLs to changelog/diff?
- Allow to escape `|` in filter argument
- sort filter: make smarter? natural sort?
- Some kind of cache to better handle multiple invocations
- HTTP service to run pipelines?

package all

import (
	"github.com/wader/bump/internal/filter"
	"github.com/wader/bump/internal/filter/docker"
	"github.com/wader/bump/internal/filter/err"
	"github.com/wader/bump/internal/filter/fetch"
	"github.com/wader/bump/internal/filter/git"
	"github.com/wader/bump/internal/filter/gitrefs"
	"github.com/wader/bump/internal/filter/key"
	"github.com/wader/bump/internal/filter/re"
	"github.com/wader/bump/internal/filter/semver"
	"github.com/wader/bump/internal/filter/sort"
	"github.com/wader/bump/internal/filter/static"
	"github.com/wader/bump/internal/filter/svn"
)

// Filters return all filters
func Filters() []filter.NamedFilter {
	return []filter.NamedFilter{
		{Name: git.Name, Help: git.Help, NewFn: git.New}, // before fetch to let it get URLs ending with .git
		{Name: gitrefs.Name, Help: gitrefs.Help, NewFn: gitrefs.New},
		{Name: docker.Name, Help: docker.Help, NewFn: docker.New},
		{Name: svn.Name, Help: svn.Help, NewFn: svn.New},
		{Name: fetch.Name, Help: fetch.Help, NewFn: fetch.New},
		{Name: semver.Name, Help: semver.Help, NewFn: semver.New},
		{Name: re.Name, Help: re.Help, NewFn: re.New},
		{Name: sort.Name, Help: sort.Help, NewFn: sort.New},
		{Name: key.Name, Help: key.Help, NewFn: key.New},
		{Name: static.Name, Help: static.Help, NewFn: static.New},
		{Name: err.Name, Help: err.Help, NewFn: err.New},
	}
}

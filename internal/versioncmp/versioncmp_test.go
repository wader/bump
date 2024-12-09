package versioncmp_test

import (
	"log"
	"sort"
	"testing"

	"github.com/wader/bump/internal/versioncmp"
)

func TestCmp(t *testing.T) {

	// s := []string{"ab.22.cc", "ab.11.dd", "ab.11.dd"}

	s := []string{"1_9_13p2", "1_9_13", "1_9_11"}

	log.Printf("b: %#+v\n", s)

	sort.SliceStable(s, func(i, j int) bool {
		return versioncmp.Cmp(s[i], s[j])
	})

	log.Printf("a: %#+v\n", s)
}

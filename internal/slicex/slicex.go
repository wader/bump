// slicex is package with generic slice functions
package slicex

func Map[F, T any](s []F, fn func(F) T) []T {
	ts := make([]T, len(s))
	for i, e := range s {
		ts[i] = fn(e)
	}
	return ts
}

func Unique[T comparable](s []T) []T {
	seen := map[T]struct{}{}
	var us []T
	for _, e := range s {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		us = append(us, e)
	}
	return us
}

// Package naivediff generates a unified diff between two texts wih the same amount of lines
// WARNING: Is naive because it assumes source and destination text have the same number of lines.
// Should only be used to make changes not to lines, not add or delete lines.
package naivediff

import (
	"fmt"
	"strings"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Diff generates a unified diff with changes to get from a to b
// Assumes a and b have the same number of lines. Panics otherwise.
func Diff(srcText, dstText string, contextLines int) string {
	srcLines := strings.Split(srcText, "\n")
	dstLines := strings.Split(dstText, "\n")

	if len(srcLines) != len(dstLines) {
		panic("src and dst needs to have same number of lines")
	}

	// TODO: diff seems to ignore last line if empty, correct?
	if srcLines[len(srcLines)-1] == "" && dstLines[len(dstLines)-1] == "" {
		srcLines = srcLines[0 : len(srcLines)-1]
		dstLines = dstLines[0 : len(dstLines)-1]
	}
	maxLines := len(srcLines)
	lastLineHasNewLine := len(srcText) > 0 && srcText[len(srcText)-1] == "\n"[0]
	const noNewLineAtEndOfFile = `\ No newline at end of file`

	type diff struct {
		start int
		stop  int
	}

	type hunk struct {
		contextStart int
		diffs        []diff
		contextStop  int
	}

	// collect ranges of lines that differ
	// each diff gets it's own hunk
	var hunks []hunk
	start := 0
	inHunk := false
	pContextStop := 0
	for i := range srcLines {
		if inHunk {
			if srcLines[i] == dstLines[i] {
				inHunk = false
				contextStart := max(max(0, start-contextLines), pContextStop)
				contextStop := min(maxLines, i+contextLines)

				hunks = append(hunks, hunk{
					contextStart: contextStart,
					diffs:        []diff{{start: start, stop: i}},
					contextStop:  contextStop,
				})

				pContextStop = contextStop
			}
		} else {
			if srcLines[i] != dstLines[i] {
				start = i
				inHunk = true
			}
		}
	}
	if inHunk {
		contextStart := max(max(0, start-contextLines), pContextStop)
		hunks = append(hunks, hunk{
			contextStart: contextStart,
			diffs:        []diff{{start: start, stop: maxLines}},
			contextStop:  maxLines,
		})
	}

	// merge hunks with overlapping context
	var mergedHunks []hunk
	mh := hunks[0]
	for _, h := range hunks[1:] {
		if mh.contextStop >= h.contextStart {
			mh.diffs = append(mh.diffs, h.diffs...)
			mh.contextStop = h.contextStop
		} else {
			mergedHunks = append(mergedHunks, mh)
			mh = h
		}
	}
	mergedHunks = append(mergedHunks, mh)

	var diffLines []string
	for _, h := range mergedHunks {
		hunkStart := h.contextStart + 1
		hunkLines := h.contextStop - h.contextStart
		if hunkLines == 1 {
			diffLines = append(diffLines,
				fmt.Sprintf("@@ -%d +%d @@", hunkStart, hunkStart))
		} else {
			diffLines = append(diffLines,
				fmt.Sprintf("@@ -%d,%d +%d,%d @@",
					hunkStart, hunkLines,
					hunkStart, hunkLines))
		}

		for i := h.contextStart; i < h.diffs[0].start; i++ {
			diffLines = append(diffLines, " "+srcLines[i])
		}
		for i, d := range h.diffs {
			if i > 0 {
				for i := h.diffs[i-1].stop; i < d.start; i++ {
					diffLines = append(diffLines, " "+srcLines[i])
				}
			}
			for i := d.start; i < d.stop; i++ {
				diffLines = append(diffLines, "-"+srcLines[i])
				if i == maxLines-1 && !lastLineHasNewLine {
					diffLines = append(diffLines, noNewLineAtEndOfFile)
				}
			}
			for i := d.start; i < d.stop; i++ {
				diffLines = append(diffLines, "+"+dstLines[i])
				if i == maxLines-1 && !lastLineHasNewLine {
					diffLines = append(diffLines, noNewLineAtEndOfFile)
				}
			}
		}
		for i := h.diffs[len(h.diffs)-1].stop; i < h.contextStop; i++ {
			diffLines = append(diffLines, " "+srcLines[i])
			if i == maxLines-1 && !lastLineHasNewLine {
				diffLines = append(diffLines, noNewLineAtEndOfFile)
			}
		}
	}
	// make sure last line has a newline
	diffLines = append(diffLines, "")

	return strings.Join(diffLines, "\n")
}

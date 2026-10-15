package corpus

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MarkerKind is the sort of claim a marker comment makes.
type MarkerKind string

const (
	// MarkerExpect anchors a seeded flaw that must be detected.
	MarkerExpect MarkerKind = "expect"

	// MarkerSafe anchors correct authorization. A finding here is a false
	// positive.
	MarkerSafe MarkerKind = "safe"

	// MarkerUnresolved anchors a site analysis is expected to give up on.
	MarkerUnresolved MarkerKind = "unresolved"
)

// Marker is a parsed marker comment from corpus source.
type Marker struct {
	Kind  MarkerKind
	ID    string
	Attrs map[string]string
	File  string
	Line  int
}

const markerPrefix = "pathwise:"

// ScanMarkers walks dir for Go files and extracts marker comments.
//
// Markers are parsed from comment text by hand rather than through go/ast so
// that the corpus can be validated even when the app does not compile — which
// is precisely the state a corpus app is in while being written.
func ScanMarkers(dir string) ([]Marker, error) {
	var markers []Marker

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "vendor" || name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		found, err := scanFile(path)
		if err != nil {
			return err
		}
		markers = append(markers, found...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan markers in %s: %w", dir, err)
	}
	return markers, nil
}

func scanFile(path string) ([]Marker, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Marker
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()

		idx := strings.Index(text, markerPrefix)
		if idx < 0 {
			continue
		}
		// Only treat it as a marker if it sits inside a comment.
		if c := strings.Index(text, "//"); c < 0 || c > idx {
			continue
		}

		m, ok := parseMarker(text[idx+len(markerPrefix):])
		if !ok {
			continue
		}
		m.File, m.Line = path, line
		out = append(out, m)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return out, nil
}

// parseMarker reads "expect kind=x id=y" into a Marker.
func parseMarker(s string) (Marker, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return Marker{}, false
	}

	kind := MarkerKind(fields[0])
	switch kind {
	case MarkerExpect, MarkerSafe, MarkerUnresolved:
	default:
		return Marker{}, false
	}

	m := Marker{Kind: kind, Attrs: map[string]string{}}
	for _, f := range fields[1:] {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			continue
		}
		if k == "id" {
			m.ID = v
			continue
		}
		m.Attrs[k] = v
	}
	return m, true
}

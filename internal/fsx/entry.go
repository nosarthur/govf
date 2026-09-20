// Package fsx: directory listing, sorting, file ops.
package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is one dir listing row.
type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	IsLink  bool
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
}

// SortKey selects sort field.
type SortKey int

const (
	SortName SortKey = iota
	SortSize
	SortTime
)

func (k SortKey) String() string {
	switch k {
	case SortSize:
		return "size"
	case SortTime:
		return "time"
	default:
		return "name"
	}
}

// ParseSortKey maps user text to SortKey.
func ParseSortKey(s string) (SortKey, bool) {
	switch strings.ToLower(s) {
	case "n", "name":
		return SortName, true
	case "s", "size":
		return SortSize, true
	case "t", "time", "mtime":
		return SortTime, true
	}
	return SortName, false
}

// SortSpec: key + direction.
type SortSpec struct {
	Key     SortKey
	Reverse bool
}

func (s SortSpec) String() string {
	if s.Reverse {
		return "-" + s.Key.String()
	}
	return "+" + s.Key.String()
}

// ReadDir lists dir; dotfiles skipped unless showHidden. Dirs always first.
func ReadDir(dir string, showHidden bool, spec SortSpec) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, de := range des {
		name := de.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		p := filepath.Join(dir, name)
		e := Entry{Name: name, Path: p}
		// lstat for link flag, stat for target type.
		if li, err := os.Lstat(p); err == nil {
			e.Mode = li.Mode()
			e.IsLink = li.Mode()&os.ModeSymlink != 0
			e.Size = li.Size()
			e.ModTime = li.ModTime()
			e.IsDir = li.IsDir()
		}
		if e.IsLink {
			if si, err := os.Stat(p); err == nil {
				e.IsDir = si.IsDir()
				e.Size = si.Size()
			}
		}
		out = append(out, e)
	}
	Sort(out, spec)
	return out, nil
}

// Sort orders entries in place: dirs first, then by spec.
func Sort(es []Entry, spec SortSpec) {
	less := func(a, b Entry) bool {
		switch spec.Key {
		case SortSize:
			if a.Size != b.Size {
				return a.Size < b.Size
			}
		case SortTime:
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.Before(b.ModTime)
			}
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		if spec.Reverse {
			return less(b, a)
		}
		return less(a, b)
	})
}

// HumanSize formats bytes compactly.
func HumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	s := "KMGTPE"[exp : exp+1]
	if v < 10 {
		return fmt.Sprintf("%.1f%s", v, s)
	}
	return fmt.Sprintf("%.0f%s", v, s)
}

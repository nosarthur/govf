package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nosarthur/govf/internal/fsx"
)

// TestMain: name order in tests; time order is the runtime default.
func TestMain(m *testing.M) {
	DefaultSort = fsx.SortSpec{Key: fsx.SortName}
	os.Exit(m.Run())
}

func TestDefaultSortIsTime(t *testing.T) {
	d := setup(t)
	saved := DefaultSort
	DefaultSort = fsx.SortSpec{Key: fsx.SortTime, Reverse: true}
	defer func() { DefaultSort = saved }()
	os.Chtimes(filepath.Join(d, "alpha.txt"), time.Unix(1e9, 0), time.Unix(1e9, 0))
	os.Chtimes(filepath.Join(d, "beta.txt"), time.Unix(2e9, 0), time.Unix(2e9, 0))
	p := NewPanel(d)
	if p.Sort != DefaultSort || p.Entries[2].Name != "beta.txt" || p.Entries[3].Name != "alpha.txt" {
		t.Errorf("default sort: %v %v", p.Sort, names(p.Entries))
	}
}

func names(es []fsx.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func setup(t *testing.T) string {
	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "dir1"), 0o755)
	os.Mkdir(filepath.Join(d, "dir2"), 0o755)
	os.WriteFile(filepath.Join(d, "alpha.txt"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(d, "beta.txt"), []byte("bb"), 0o644)
	os.WriteFile(filepath.Join(d, ".dot"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(d, "dir1", "inner"), []byte("i"), 0o644)
	return d
}

func TestPanelNav(t *testing.T) {
	d := setup(t)
	p := NewPanel(d)
	if len(p.Entries) != 4 {
		t.Fatalf("entries %d", len(p.Entries))
	}
	p.Move(10)
	if p.Cur != 3 {
		t.Errorf("clamp: %d", p.Cur)
	}
	p.Move(-10)
	if p.Cur != 0 || p.Current().Name != "dir1" {
		t.Errorf("top: %d", p.Cur)
	}
	if !p.Enter() || p.Dir != filepath.Join(d, "dir1") || p.Current().Name != "inner" {
		t.Errorf("enter: %s", p.Dir)
	}
	if p.Enter() {
		t.Error("entered a file")
	}
	p.Up()
	if p.Dir != d || p.Current().Name != "dir1" {
		t.Errorf("up: %s cur=%d", p.Dir, p.Cur)
	}
	p.Move(1)
	p.Enter()
	p.Up()
	if p.Current().Name != "dir2" {
		t.Error("cursor not restored on parent")
	}
	// hidden
	p.Hidden = true
	p.Reload()
	if len(p.Entries) != 5 || p.Current().Name != "dir2" {
		t.Errorf("hidden reload: n=%d cur=%s", len(p.Entries), p.Current().Name)
	}
}

func TestPanelSelectSearch(t *testing.T) {
	d := setup(t)
	p := NewPanel(d)
	if got := p.Targets(); len(got) != 1 || got[0] != filepath.Join(d, "dir1") {
		t.Errorf("targets cursor: %v", got)
	}
	p.MoveTo(2)
	p.ToggleSel()
	p.MoveTo(3)
	p.ToggleSel()
	p.MoveTo(0)
	if got := p.Targets(); len(got) != 2 || filepath.Base(got[0]) != "alpha.txt" {
		t.Errorf("targets sel: %v", got)
	}
	p.MoveTo(3)
	p.ToggleSel()
	if len(p.Selected) != 1 {
		t.Error("untoggle")
	}
	p.Reload()
	if len(p.Selected) != 1 {
		t.Error("selection lost on reload")
	}
	p.MoveTo(0)
	if !p.Search("BETA", false) || p.Current().Name != "beta.txt" {
		t.Error("search fwd")
	}
	if !p.Search("dir", true) || p.Current().Name != "dir2" {
		t.Errorf("search back: %s", p.Current().Name)
	}
	if p.Search("zzz", false) {
		t.Error("search miss")
	}
	p.Cur = 9
	p.Scroll(3)
	if p.Top != 7 {
		t.Errorf("scroll top %d", p.Top)
	}
	p.Cur = 2
	p.Scroll(3)
	if p.Top != 2 {
		t.Errorf("scroll up top %d", p.Top)
	}
}

package fsx

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mk(t *testing.T, dir, name string, size int, mt time.Time) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func names(es []Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestReadDirSort(t *testing.T) {
	d := t.TempDir()
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mk(t, d, "b.txt", 30, base.Add(2*time.Hour))
	mk(t, d, "a.txt", 10, base.Add(3*time.Hour))
	mk(t, d, "C.txt", 20, base.Add(1*time.Hour))
	mk(t, d, ".hid", 1, base)
	os.Mkdir(filepath.Join(d, "zdir"), 0o755)

	cases := []struct {
		spec   SortSpec
		hidden bool
		want   []string
	}{
		{SortSpec{SortName, false}, false, []string{"zdir", "a.txt", "b.txt", "C.txt"}},
		{SortSpec{SortName, true}, false, []string{"zdir", "C.txt", "b.txt", "a.txt"}},
		{SortSpec{SortSize, false}, false, []string{"zdir", "a.txt", "C.txt", "b.txt"}},
		{SortSpec{SortSize, true}, false, []string{"zdir", "b.txt", "C.txt", "a.txt"}},
		{SortSpec{SortTime, false}, false, []string{"zdir", "C.txt", "b.txt", "a.txt"}},
		{SortSpec{SortTime, true}, false, []string{"zdir", "a.txt", "b.txt", "C.txt"}},
		{SortSpec{SortName, false}, true, []string{"zdir", ".hid", "a.txt", "b.txt", "C.txt"}},
	}
	for _, c := range cases {
		es, err := ReadDir(d, c.hidden, c.spec)
		if err != nil {
			t.Fatal(err)
		}
		if got := names(es); !eq(got, c.want) {
			t.Errorf("%v hidden=%v: got %v want %v", c.spec, c.hidden, got, c.want)
		}
	}
}

func TestParseSortKey(t *testing.T) {
	for in, want := range map[string]SortKey{"n": SortName, "size": SortSize, "T": SortTime} {
		k, ok := ParseSortKey(in)
		if !ok || k != want {
			t.Errorf("%q: got %v,%v", in, k, ok)
		}
	}
	if _, ok := ParseSortKey("bogus"); ok {
		t.Error("bogus accepted")
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0B", 512: "512B", 1024: "1.0K", 1536: "1.5K", 10 * 1024: "10K", 3 * 1024 * 1024: "3.0M"} {
		if got := HumanSize(n); got != want {
			t.Errorf("%d: got %s want %s", n, got, want)
		}
	}
}

func TestCopyMoveDelete(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "f"), []byte("hi"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "g"), []byte("yo"), 0o600)
	os.Symlink("f", filepath.Join(src, "ln"))
	dst := filepath.Join(d, "dst")
	os.Mkdir(dst, 0o755)

	p, err := Copy(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(p, "sub", "g")); string(b) != "yo" {
		t.Fatal("copy content mismatch")
	}
	if l, err := os.Readlink(filepath.Join(p, "ln")); err != nil || l != "f" {
		t.Fatal("symlink not preserved")
	}
	p2, err := Copy(src, dst)
	if err != nil || filepath.Base(p2) != "src_1" {
		t.Fatalf("unique dest: %v %v", p2, err)
	}
	if _, err := Copy(src, src); err == nil {
		t.Fatal("copy into self allowed")
	}
	if _, err := Copy(src, filepath.Join(src, "sub")); err == nil {
		t.Fatal("copy into subdir of self allowed")
	}
	m, err := Move(filepath.Join(src, "f"), dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(src, "f")); !os.IsNotExist(err) {
		t.Fatal("src still exists after move")
	}
	if b, _ := os.ReadFile(m); string(b) != "hi" {
		t.Fatal("moved content mismatch")
	}
	if err := Rename(m, p2); err == nil {
		t.Fatal("rename clobbered")
	}
	if err := MoveTo(m, p2); err == nil {
		t.Fatal("MoveTo clobbered")
	}
	deep := filepath.Join(d, "new", "deep", "f")
	if err := MoveTo(m, deep); err != nil {
		t.Fatal(err)
	}
	if err := MoveTo(deep, m); err != nil {
		t.Fatal(err)
	}
	if err := Rename(m, filepath.Join(dst, "renamed")); err != nil {
		t.Fatal(err)
	}
	if err := Delete(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("delete failed")
	}
	if err := Touch(filepath.Join(dst, "renamed")); err == nil {
		t.Fatal("touch clobbered")
	}
}

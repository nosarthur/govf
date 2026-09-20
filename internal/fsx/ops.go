package fsx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Mkdir creates dir (and parents).
func Mkdir(path string) error { return os.MkdirAll(path, 0o755) }

// Touch creates empty file; errors if exists.
func Touch(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

// Rename renames; refuses to clobber.
func Rename(oldPath, newPath string) error {
	if _, err := os.Lstat(newPath); err == nil {
		return fmt.Errorf("%s exists", filepath.Base(newPath))
	}
	return os.Rename(oldPath, newPath)
}

// Delete removes file or dir tree.
func Delete(path string) error { return os.RemoveAll(path) }

// UniqueDest returns dst if free, else dst_1, dst_2...
func UniqueDest(dst string) string {
	if _, err := os.Lstat(dst); err != nil {
		return dst
	}
	ext := filepath.Ext(dst)
	base := dst[:len(dst)-len(ext)]
	for i := 1; ; i++ {
		c := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Lstat(c); err != nil {
			return c
		}
	}
}

// Copy copies src (file/dir/symlink) into dstDir; returns final path.
func Copy(src, dstDir string) (string, error) {
	dst := UniqueDest(filepath.Join(dstDir, filepath.Base(src)))
	if inside(src, dst) {
		return "", errors.New("cannot copy dir into itself")
	}
	return dst, copyPath(src, dst)
}

// Move moves src into dstDir; falls back to copy+delete across devices.
func Move(src, dstDir string) (string, error) {
	dst := UniqueDest(filepath.Join(dstDir, filepath.Base(src)))
	return dst, MoveTo(src, dst)
}

// MoveTo moves src to exact path dst; refuses to clobber; creates parent.
func MoveTo(src, dst string) error {
	if inside(src, dst) {
		return errors.New("cannot move dir into itself")
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s exists", dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyPath(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// inside: child is parent or under it.
func inside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || !(rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func copyPath(src, dst string) error {
	li, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case li.Mode()&os.ModeSymlink != 0:
		t, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(t, dst)
	case li.IsDir():
		if err := os.MkdirAll(dst, li.Mode().Perm()); err != nil {
			return err
		}
		des, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, de := range des {
			if err := copyPath(filepath.Join(src, de.Name()), filepath.Join(dst, de.Name())); err != nil {
				return err
			}
		}
		return nil
	default:
		return copyFile(src, dst, li.Mode().Perm())
	}
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

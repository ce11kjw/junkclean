package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirSizeAndClean(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a"), []byte("12345"), 0644)
	os.MkdirAll(filepath.Join(root, "sub"), 0755)
	os.WriteFile(filepath.Join(root, "sub", "b"), []byte("123"), 0644)

	sz, n := dirSize(root)
	if sz != 8 || n != 2 {
		t.Fatalf("dirSize got %d/%d, want 8/2", sz, n)
	}

	scanSt.categories = []Category{{Items: []JunkItem{
		{ID: "cache:x", Path: filepath.Join(root, "sub"), Size: 3, Count: 1},
	}}}
	freed, errs := scanSt.clean([]string{"cache:x"})
	if len(errs) != 0 || freed != 3 {
		t.Fatalf("clean got %d/%v, want 3/[]", freed, errs)
	}
	if _, err := os.Stat(filepath.Join(root, "sub")); !os.IsNotExist(err) {
		t.Fatal("expected sub dir removed")
	}
}

package res

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLooseResourcesMatchCaseInsensitivePaths(t *testing.T) {
	manager := &Manager{Root: t.TempDir(), Archives: []*GRF{resourceAliasTestArchive(t, map[string][]byte{
		"data/source.gat": []byte("archive"),
	})}}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{
		"Data/ResNameTable.TXT": []byte("clone.gat#SOURCE.GAT#\n유저인터페이스\\map\\clone.bmp#유저인터페이스\\MAP\\Source.BMP#\n"),
		"Data/Source.Gat":       []byte("loose"),
		"Data/Texture/유저인터페이스/Map/source.bmp": []byte("minimap"),
	})
	for name, want := range map[string]string{
		`data\clone.gat`:  "loose",
		"DATA/SOURCE.GAT": "loose",
		"data/texture/유저인터페이스/map/clone.bmp": "minimap",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for range 5 {
				got, err := manager.ReadFileExact(name)
				if err != nil || string(got) != want || !manager.HasFileExact(name) {
					t.Fatalf("read %q = %q, %v; want %q", name, got, err, want)
				}
			}
		})
	}
}

func TestLooseResourceExactSpellingWins(t *testing.T) {
	manager := &Manager{Root: t.TempDir()}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"Source.gat": []byte("upper")})
	if _, exists := manager.Find("SOURCE.gat"); !exists {
		t.Fatal("case-insensitive fallback did not find Source.gat")
	}
	if _, err := os.Stat(filepath.Join(manager.Root, "source.gat")); err == nil {
		t.Skip("filesystem cannot store names differing only by case")
	}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"source.gat": []byte("lower")})
	for name, want := range map[string]string{"source.gat": "lower", "Source.gat": "upper"} {
		if got, err := manager.ReadFileExact(name); err != nil || string(got) != want {
			t.Fatalf("exact spelling %s = %q, %v", name, got, err)
		}
	}
}

func TestLooseDirectoryCacheRefreshes(t *testing.T) {
	manager := &Manager{Root: t.TempDir()}
	if _, ok := manager.Find("new.gat"); ok {
		t.Fatal("unexpected resource before creation")
	}
	info, err := os.Stat(manager.Root)
	if err != nil {
		t.Fatal(err)
	}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"NEW.GAT": []byte("new")})
	// Ensure the directory timestamp changes even on coarse-resolution filesystems.
	modified := info.ModTime().Add(2 * time.Second)
	if err := os.Chtimes(manager.Root, modified, modified); err != nil {
		t.Fatal(err)
	}
	if got, err := manager.ReadFileExact("new.gat"); err != nil || string(got) != "new" {
		t.Fatalf("cached directory missed the new file: %q, %v", got, err)
	}
}

package res

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/korean"
)

func TestManagerResourceAliases(t *testing.T) {
	const table = "// Clone resources\r\n" +
		"NEW_1-1.gat#New_Zone01.gat#\r\n" +
		"new_1-1.gnd#New_Zone01.gnd#\r\n" +
		"new_1-1.rsw#New_Zone01.rsw# // comment\r\n" +
		"유저인터페이스\\map\\new_1-1.bmp#유저인터페이스\\map\\New_Zone01.bmp#\r\n"
	for _, encoding := range []string{"utf8", "euc-kr"} {
		for _, storage := range []string{"loose", "grf"} {
			t.Run(encoding+"/"+storage, func(t *testing.T) {
				data := []byte("\ufeff" + table)
				if encoding == "euc-kr" {
					var err error
					data, err = korean.EUCKR.NewEncoder().Bytes([]byte(table))
					if err != nil {
						t.Fatal(err)
					}
				}
				files := map[string][]byte{
					"data/resnametable.txt": data,
					"data/New_Zone01.gat":   []byte("collision"),
					"data/New_Zone01.gnd":   []byte("ground"),
					"data/New_Zone01.rsw":   []byte("world"),
					"data/texture/유저인터페이스/map/New_Zone01.bmp": []byte("minimap"),
				}
				manager := &Manager{Root: t.TempDir()}
				if storage == "grf" {
					manager.Archives = []*GRF{resourceAliasTestArchive(t, files)}
				} else {
					writeResourceAliasTestFiles(t, manager.Root, files)
				}
				requests := map[string]string{
					`data\NEW_1-1.gat`: "collision",
					"data/new_1-1.gnd": "ground",
					"data/new_1-1.rsw": "world",
					"data/texture/유저인터페이스/map/new_1-1.bmp": "minimap",
				}
				for name, want := range requests {
					for _, read := range []func(string) ([]byte, error){manager.ReadFile, manager.ReadFileExact} {
						got, err := read(name)
						if err != nil || string(got) != want {
							t.Fatalf("read(%q) = %q, %v; want %q", name, got, err, want)
						}
					}
					if !manager.HasFileExact(name) {
						t.Fatalf("HasFileExact(%q) missed alias", name)
					}
				}
				// Model/texture loaders can also pass raw EUC-KR filenames.
				rawName, err := korean.EUCKR.NewEncoder().String("data\\texture\\유저인터페이스\\map\\new_1-1.bmp")
				if err != nil {
					t.Fatal(err)
				}
				if got, err := manager.ReadFileExact(rawName); err != nil || string(got) != "minimap" {
					t.Fatalf("read EUC-KR path = %q, %v", got, err)
				}
			})
		}
	}
}

func TestManagerResourceAliasPriority(t *testing.T) {
	manager := &Manager{Root: t.TempDir(), Archives: []*GRF{
		resourceAliasTestArchive(t, map[string][]byte{
			"data/resnametable.txt": []byte("a.gat#patch.gat#\nb.gat#patch.gat#\n"),
			"data/patch.gat":        []byte("patch"),
			"data/direct.gat":       []byte("direct archive resource"),
		}),
		resourceAliasTestArchive(t, map[string][]byte{
			"data/resnametable.txt": []byte("a.gat#base.gat#\nb.gat#base.gat#\nc.gat#base.gat#\n"),
			"data/base.gat":         []byte("base"),
		}),
	}}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{
		"data/resnametable.txt": []byte("a.gat#loose.gat#\ndirect.gat#loose.gat#\n"),
		"data/loose.gat":        []byte("loose"),
	})
	for name, want := range map[string]string{
		"a.gat": "loose", "b.gat": "patch", "c.gat": "base", "direct.gat": "direct archive resource",
	} {
		if got, err := manager.ReadFileExact("data/" + name); err != nil || string(got) != want {
			t.Fatalf("read %s = %q, %v; want %q", name, got, err, want)
		}
	}
	// An archive alias may point to a loose file, and a direct loose file
	// still overrides that alias after the table has been cached.
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"data/patch.gat": []byte("loose target")})
	if got, err := manager.ReadFile("data/b.gat"); err != nil || string(got) != "loose target" {
		t.Fatalf("loose alias target = %q, %v", got, err)
	}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"data/b.gat": []byte("direct loose resource")})
	if got, err := manager.ReadFile("data/b.gat"); err != nil || string(got) != "direct loose resource" {
		t.Fatalf("direct loose resource = %q, %v", got, err)
	}
}

func TestManagerResourceAliasMissingAndExactPaths(t *testing.T) {
	manager := &Manager{Root: t.TempDir(), Archives: []*GRF{resourceAliasTestArchive(t, map[string][]byte{
		"data/resnametable.txt": []byte("clone.gat#source.gat#\n" +
			"missing.gat#absent.gat#\nself.gat#self.gat#\na.gat#b.gat#\nb.gat#a.gat#\n" +
			"invalid.gat#../outside.gat#\nincomplete.gat#source.gat\n"),
		"data/source.gat": []byte("source"),
	})}}
	for _, name := range []string{"data/missing.gat", "data/self.gat", "data/a.gat", "data/invalid.gat", "data/incomplete.gat", "data/other/clone.gat", "clone.gat"} {
		if _, err := manager.ReadFileExact(name); !errors.Is(err, errResourceNotFound) {
			t.Fatalf("ReadFileExact(%q) = %v, want missing resource", name, err)
		}
		if manager.HasFileExact(name) {
			t.Fatalf("HasFileExact(%q) matched a missing or suffix-only alias", name)
		}
	}
	if got, err := manager.ReadFile("clone.gat"); err != nil || string(got) != "source" {
		t.Fatalf("legacy suffix lookup of alias target = %q, %v", got, err)
	}
	// No suffix fallback may be introduced for aliased sounds/images either.
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{
		"data/resnametable.txt": []byte("sound.wav#effect/source.wav#\n"),
	})
	other := &Manager{Root: manager.Root, Archives: []*GRF{resourceAliasTestArchive(t, map[string][]byte{
		"data/wav/effect/source.wav": []byte("sound"),
	})}}
	if _, err := other.ReadFileExact("sound.wav"); !errors.Is(err, errResourceNotFound) || other.HasFileExact("sound.wav") {
		t.Fatalf("exact alias lookup used a suffix: %v", err)
	}
	if got, err := other.ReadFile("sound.wav"); err != nil || string(got) != "sound" {
		t.Fatalf("legacy alias suffix lookup = %q, %v", got, err)
	}
}

func TestManagerResourceAliasPreservesReadErrors(t *testing.T) {
	manager := &Manager{Root: t.TempDir()}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{
		"data/resnametable.txt": []byte("clone.gat#source.gat#\n"),
		"data/clone.gat":        []byte("unreadable"),
		"data/source.gat":       []byte("readable"),
	})
	name := filepath.Join(manager.Root, "data", "clone.gat")
	if err := os.Chmod(name, 0000); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(name); err == nil {
		t.Skip("current user can read files without read permission")
	}
	for _, read := range []func(string) ([]byte, error){manager.ReadFile, manager.ReadFileExact} {
		if _, err := read("data/clone.gat"); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("read error replaced by alias: %v", err)
		}
	}
}

func writeResourceAliasTestFiles(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for name, data := range files {
		filename := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func resourceAliasTestArchive(t *testing.T, files map[string][]byte) *GRF {
	t.Helper()
	root := t.TempDir()
	writeResourceAliasTestFiles(t, root, files)
	filename := filepath.Join(t.TempDir(), "data.grf")
	if _, err := PackGRF(filename, root); err != nil {
		t.Fatal(err)
	}
	archive, err := OpenGRF(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = archive.Close() })
	return archive
}

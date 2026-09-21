package res

import (
	"bytes"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/bmp"
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
	archives := []*GRF{
		resourceAliasTestArchive(t, map[string][]byte{
			"data/resnametable.txt": []byte("a.gat#patch.gat#\na.gat#base.gat#\nb.gat#patch.gat#\n"),
			"data/patch.gat":        []byte("patch"),
			"data/direct.gat":       []byte("direct archive resource"),
		}),
		resourceAliasTestArchive(t, map[string][]byte{
			"data/resnametable.txt": []byte("a.gat#base.gat#\nc.gat#base.gat#\n"),
			"data/base.gat":         []byte("base"),
		}),
	}
	manager := &Manager{Root: t.TempDir(), Archives: archives}
	for _, name := range []string{"a.gat", "b.gat"} {
		if got, err := manager.ReadFileExact("data/" + name); err != nil || string(got) != "patch" {
			t.Fatalf("archive priority/first definition: %s = %q, %v", name, got, err)
		}
	}
	if manager.HasFileExact("data/c.gat") {
		t.Fatal("lower-priority table restored a removed alias")
	}
	// An archived table may point to a loose target.
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"data/patch.gat": []byte("loose target")})
	if got, err := manager.ReadFile("data/b.gat"); err != nil || string(got) != "loose target" {
		t.Fatalf("loose alias target = %q, %v", got, err)
	}
	// A new manager loads the loose replacement table as a whole.
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{
		"data/resnametable.txt": []byte("a.gat#loose.gat#\ndirect.gat#loose.gat#\n"),
		"data/loose.gat":        []byte("loose"),
	})
	manager = &Manager{Root: manager.Root, Archives: archives}
	for name, want := range map[string]string{"a.gat": "loose", "direct.gat": "direct archive resource"} {
		if got, err := manager.ReadFileExact("data/" + name); err != nil || string(got) != want {
			t.Fatalf("loose table priority: %s = %q, %v; want %q", name, got, err, want)
		}
	}
	if manager.HasFileExact("data/b.gat") {
		t.Fatal("archive table filled a gap in the loose replacement")
	}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"data/a.gat": []byte("direct loose resource")})
	if got, err := manager.ReadFile("data/a.gat"); err != nil || string(got) != "direct loose resource" {
		t.Fatalf("direct loose override = %q, %v", got, err)
	}
	writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{"data/resnametable.txt": []byte("// No aliases for this client\n")})
	manager = &Manager{Root: manager.Root, Archives: archives}
	if !manager.HasFileExact("data/direct.gat") || manager.HasFileExact("data/b.gat") {
		t.Fatal("empty table did not disable aliases while preserving direct resources")
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
	if _, _, err := manager.ReadFileCandidates([]string{"data/clone.gat", "clone.gat"}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("candidate lookup replaced the direct read error: %v", err)
	}
}

func TestResourceAliasAfterGRFExtraction(t *testing.T) {
	archive := resourceAliasTestArchive(t, map[string][]byte{
		"data/resnametable.txt": []byte("clone.gat#Source.gat#\n"),
		"data/Source.gat":       []byte("source"),
	})
	// Match grf-extract: archive.Names returns normalized lowercase paths.
	root := t.TempDir()
	for _, name := range archive.Names() {
		data, err := archive.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		writeResourceAliasTestFiles(t, root, map[string][]byte{name: data})
	}
	manager := &Manager{Root: root}
	if got, err := manager.ReadFileExact("data/clone.gat"); err != nil || string(got) != "source" {
		t.Fatalf("extracted alias = %q, %v", got, err)
	}
	if !manager.HasFileExact("data/clone.gat") {
		t.Fatal("HasFileExact missed the extracted alias target")
	}
}

func TestResourceCandidatesPreferDirectMapFiles(t *testing.T) {
	for _, ext := range []string{"gat", "gnd", "rsw"} {
		t.Run(ext, func(t *testing.T) {
			name := "new_1-1." + ext
			source := "new_zone01." + ext
			manager := &Manager{Root: t.TempDir(), Archives: []*GRF{resourceAliasTestArchive(t, map[string][]byte{
				"data/resnametable.txt": []byte(name + "#" + source + "#\n"),
				"data/" + source:        []byte("alias"),
			})}}
			writeResourceAliasTestFiles(t, manager.Root, map[string][]byte{name: []byte("direct")})
			candidates := []string{`data\` + name, "data/" + name, name}
			got, selected, err := manager.ReadFileCandidates(candidates)
			if err != nil || string(got) != "direct" || selected != name {
				t.Fatalf("direct lookup = %q, %q, %v", got, selected, err)
			}
			if path, got, ok := manager.ReadFirst(candidates); !ok || string(got) != "direct" || path != filepath.Join(manager.Root, name) {
				t.Fatalf("ReadFirst = %q, %q, %v", path, got, ok)
			}
			if err := os.Remove(filepath.Join(manager.Root, name)); err != nil {
				t.Fatal(err)
			}
			if got, _, err := manager.ReadFileCandidates(candidates); err != nil || string(got) != "alias" {
				t.Fatalf("alias fallback = %q, %v", got, err)
			}
		})
	}
}

func TestResourceExactPathsPrecedeArchiveSuffixes(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(map[bool]string{false: "alias", true: "direct"}[direct], func(t *testing.T) {
			files := map[string][]byte{"data/source.gat": []byte("exact alias")}
			want := "exact alias"
			if direct {
				files["data/clone.gat"] = []byte("exact direct")
				want = "exact direct"
			}
			manager := &Manager{Root: t.TempDir(), Archives: []*GRF{
				resourceAliasTestArchive(t, map[string][]byte{
					"data/resnametable.txt":  []byte("clone.gat#source.gat#\n"),
					"backup/data/clone.gat":  []byte("direct suffix"),
					"backup/data/source.gat": []byte("alias suffix"),
				}),
				resourceAliasTestArchive(t, files),
			}}
			for _, read := range []func(string) ([]byte, error){manager.ReadFile, manager.ReadFileExact} {
				if got, err := read("data/clone.gat"); err != nil || string(got) != want {
					t.Fatalf("read = %q, %v; want %q", got, err, want)
				}
			}
			if got, _, err := manager.ReadFileCandidates([]string{"data/clone.gat", "clone.gat"}); err != nil || string(got) != want {
				t.Fatalf("candidate read = %q, %v; want %q", got, err, want)
			}
		})
	}
}

func TestMinimapAliasPrecedesUnrelatedArchiveSuffix(t *testing.T) {
	makeBMP := func(width, height int) []byte {
		var buf bytes.Buffer
		if err := bmp.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	manager := &Manager{Root: t.TempDir(), Archives: []*GRF{resourceAliasTestArchive(t, map[string][]byte{
		"data/resnametable.txt":                   []byte("유저인터페이스\\map\\new_1-1.bmp#유저인터페이스\\map\\new_zone01.bmp#\n"),
		"data/texture/유저인터페이스/map/new_zone01.bmp": makeBMP(40, 60),
		"data/texture/unrelated/new_1-1.bmp":      makeBMP(24, 24),
	})}}
	candidates := []string{"data/texture/유저인터페이스/map/new_1-1.bmp", "new_1-1.bmp"}
	for _, load := range []func(*Manager, []string) (image.Image, string, error){LoadImage, LoadImageExact} {
		img, source, err := load(manager, candidates)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 40 || img.Bounds().Dy() != 60 || source != candidates[0] {
			t.Fatalf("unrelated suffix replaced minimap alias: %v from %s", img.Bounds(), source)
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

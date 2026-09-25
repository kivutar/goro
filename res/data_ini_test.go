package res

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestParseDataINIEncodings(t *testing.T) {
	ini := "[Data]\r\n0=café.grf\r\n"
	type iniEncoding struct {
		name string
		data []byte
	}
	encodings := []iniEncoding{
		{"UTF-8", []byte(ini)},
		{"UTF-8 BOM", []byte("\ufeff" + ini)},
	}
	for _, encoding := range []struct {
		name  string
		order binary.ByteOrder
	}{
		{"UTF-16 LE BOM", binary.LittleEndian},
		{"UTF-16 BE BOM", binary.BigEndian},
	} {
		units := utf16.Encode([]rune("\ufeff" + ini))
		data := make([]byte, 2*len(units))
		for i, unit := range units {
			encoding.order.PutUint16(data[2*i:], unit)
		}
		encodings = append(encodings, iniEncoding{encoding.name, data})
	}
	for _, encoding := range encodings {
		t.Run(encoding.name, func(t *testing.T) {
			names, err := parseDataINI(encoding.data)
			if err != nil || !slices.Equal(names, []string{"café.grf"}) {
				t.Fatalf("parseDataINI = %v, %v; want [café.grf]", names, err)
			}
		})
	}
}

func TestManagerDataINIArchiveOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a-base.grf", "m-patch.gpf", "z-custom.grf", "data.grf", "event.grf"} {
		if err := writeTestGRF(filepath.Join(root, name), `data\shared.txt`, []byte(name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeTestGRF(filepath.Join(root, "fallback.grf"), `data\fallback.txt`, []byte("fallback")); err != nil {
		t.Fatal(err)
	}
	// Numeric order differs from both line order and alphabetical order.
	ini := "[Data]\n10=a-base.grf\n2=m-patch.gpf\n0=z-custom.grf\n11=fallback.grf\n12=z-custom.grf\n"
	if err := os.WriteFile(filepath.Join(root, "DATA.INI"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := openDataINITestManager(t, root)
	assertDataINIArchiveNames(t, manager, []string{"z-custom.grf", "m-patch.gpf", "a-base.grf", "fallback.grf"})
	data, err := manager.ReadFile(`data\shared.txt`)
	if err != nil || string(data) != "z-custom.grf" {
		t.Fatalf("shared resource = %q, %v; want highest-priority archive", data, err)
	}
	data, err = manager.ReadFile(`data\fallback.txt`)
	if err != nil || string(data) != "fallback" {
		t.Fatalf("fallback resource = %q, %v", data, err)
	}
	if err := os.Mkdir(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", "shared.txt"), []byte("loose override"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err = manager.ReadFile(`data\shared.txt`)
	if err != nil || string(data) != "loose override" {
		t.Fatalf("loose override = %q, %v", data, err)
	}
}

func TestManagerDataINIPathsAndFormatting(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Patches"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestGRF(filepath.Join(root, "Patches", "Custom.GRF"), "custom.txt", []byte("custom")); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external.grf")
	if err := writeTestGRF(external, "external.txt", []byte("external")); err != nil {
		t.Fatal(err)
	}
	ini := "\ufeff; client archives\r\n[Other] ; ignored section\r\nignored=value\r\n[ dAtA ]# archive priority\r\n# comment\r\n 2 = \"patches\\custom.grf\"\r\n 10 = '" + external + "'\r\n 11 = \r\n"
	if err := os.WriteFile(filepath.Join(root, "DaTa.InI"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := openDataINITestManager(t, root)
	assertDataINIArchiveNames(t, manager, []string{"Custom.GRF", "external.grf"})
	for _, name := range []string{"custom", "external"} {
		data, err := manager.ReadFileExact(name + ".txt")
		if err != nil || string(data) != name {
			t.Fatalf("read %s = %q, %v", name, data, err)
		}
	}
}

func TestManagerDefaultArchives(t *testing.T) {
	for _, tc := range []struct {
		name     string
		archives []string
		want     []string
	}{
		{"all layers", []string{"data.grf", "sdata.grf", "rdata.grf", "fdata.grf"}, []string{"fdata.grf", "rdata.grf", "sdata.grf", "data.grf"}},
		{"renewal", []string{"data.grf", "rdata.grf"}, []string{"rdata.grf", "data.grf"}},
		{"sakray", []string{"data.grf", "sdata.grf"}, []string{"sdata.grf", "data.grf"}},
		{"base only", []string{"data.grf"}, []string{"data.grf"}},
		{"case insensitive", []string{"DATA.GRF", "FDATA.GRF"}, []string{"FDATA.GRF", "DATA.GRF"}},
		{"no standard archives", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range append(slices.Clone(tc.archives), "event.grf", "00-custom.grf", "patch.gpf") {
				if err := writeTestGRF(filepath.Join(root, name), "shared.txt", []byte(name)); err != nil {
					t.Fatal(err)
				}
			}
			manager := openDataINITestManager(t, root)
			assertDataINIArchiveNames(t, manager, tc.want)
			data, err := manager.ReadFileExact("shared.txt")
			if len(tc.want) == 0 {
				if err == nil {
					t.Fatal("loaded an unlisted custom archive")
				}
			} else if err != nil || string(data) != tc.want[0] {
				t.Fatalf("shared resource = %q, %v; want %s", data, err, tc.want[0])
			}
		})
	}
}

func TestManagerEmptyDataINIDisablesDefaultArchives(t *testing.T) {
	root := t.TempDir()
	if err := writeTestGRF(filepath.Join(root, "data.grf"), "shared.txt", []byte("base")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data.ini"), []byte("[Data]\n0=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := openDataINITestManager(t, root)
	assertDataINIArchiveNames(t, manager, nil)
}

func TestManagerDataINIErrors(t *testing.T) {
	for _, tc := range []struct{ name, ini, want string }{
		{"missing section", "0=data.grf\n", "missing [Data]"},
		{"bad section", "[Data\n", "line 1"},
		{"section trailing text", "[Data] garbage\n", "invalid section header"},
		{"section extra bracket", "[Data]]\n", "invalid section header"},
		{"missing equals", "[Data]\ndata.grf\n", "line 2"},
		{"non numeric priority", "[Data]\nfirst=data.grf\n", "invalid archive priority"},
		{"negative priority", "[Data]\n-1=data.grf\n", "invalid archive priority"},
		{"duplicate priority", "[Data]\n0=data.grf\n0=event.grf\n", "duplicate archive priority"},
		{"missing archive", "[Data]\n0=missing.grf\n", "missing.grf"},
		{"corrupt archive", "[Data]\n0=broken.grf\n", "broken.grf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := writeTestGRF(filepath.Join(root, "data.grf"), "shared.txt", []byte("base")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "broken.grf"), []byte("invalid archive"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "DATA.INI"), []byte(tc.ini), 0o644); err != nil {
				t.Fatal(err)
			}
			manager, err := NewManager(root)
			if err == nil {
				for _, archive := range manager.Archives {
					_ = archive.Close()
				}
				t.Fatal("expected an error instead of silently using default archives")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v; want %q", err, tc.want)
			}
		})
	}
}

func TestManagerClosesArchivesAfterInitializationError(t *testing.T) {
	for _, tc := range []struct{ name, resource, ini string }{
		{"archive error", "shared.txt", "[Data]\n0=data.grf\n1=missing.grf\n"},
		{"client info error", "data/clientinfo.xml", "[Data]\n0=data.grf\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			archivePath := filepath.Join(root, "data.grf")
			if err := writeTestGRF(archivePath, tc.resource, []byte("<invalid")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "DATA.INI"), []byte(tc.ini), 0o644); err != nil {
				t.Fatal(err)
			}
			manager, err := NewManager(root)
			if err == nil {
				for _, archive := range manager.Archives {
					_ = archive.Close()
				}
				t.Fatal("expected initialization error")
			}
			// On Windows an open GRF handle prevents renaming the file.
			if err := os.Rename(archivePath, archivePath+".closed"); err != nil {
				t.Fatalf("archive left open after initialization failed: %v", err)
			}
		})
	}
}

func openDataINITestManager(t *testing.T, root string) *Manager {
	t.Helper()
	manager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, archive := range manager.Archives {
			if err := archive.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	return manager
}

func assertDataINIArchiveNames(t *testing.T, manager *Manager, want []string) {
	t.Helper()
	var got []string
	for _, archive := range manager.Archives {
		got = append(got, filepath.Base(archive.Path()))
	}
	if !slices.EqualFunc(got, want, strings.EqualFold) {
		t.Fatalf("archives = %v, want %v", got, want)
	}
}

package game

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/res"
	lua "github.com/yuin/gopher-lua"
)

func TestCompanionAIStandardIOWithSelectedFolder(t *testing.T) {
	for _, selected := range []bool{false, true} {
		name := "native"
		if selected {
			name = "selected"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "AI"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "AI/config.txt"), []byte("hello\nworld\n"), 0600); err != nil {
				t.Fatal(err)
			}
			var resources *res.Manager
			var err error
			cfg := config.Config{DataDir: root}
			t.Chdir(root) // Legacy Android uses the same client and app directory.
			if selected {
				cfg.DataDir, cfg.AIStateDir = t.TempDir(), t.TempDir()
				t.Chdir(cfg.DataDir)
				resources, err = res.NewManagerFS(os.DirFS(root))
			} else {
				resources, err = res.NewManager(root)
			}
			if err != nil {
				t.Fatal(err)
			}
			ai := &companionAI{state: lua.NewState(), stateDir: cfg.AIStateDir}
			defer ai.close()
			ai.registerFileIO(client.Context{Config: cfg, Resources: resources})
			if err := ai.state.DoString(`
local originalInput, originalOutput = io.input(), io.output()
local f = assert(io.open("AI/config.txt"))
assert(io.input(f) == f and io.input() == f)
assert(io.read() == "hello")
local remaining = {}
for line in io.lines() do table.insert(remaining, line) end
assert(#remaining == 1 and remaining[1] == "world")
assert(io.type(f) == "file") -- Borrowed default input stays open at EOF.
assert(f:read() == nil)
io.close(f)
assert(not pcall(io.read))

local named = io.input("AI/config.txt")
assert(io.read(5, "*l") == "hello")
assert(io.read() == "world")
io.close(named)
local lines = {}
for line in io.lines("AI/config.txt") do table.insert(lines, line) end
assert(#lines == 2 and lines[1] == "hello" and lines[2] == "world")

local saved = assert(io.open("AI/saved.txt", "w+"))
assert(io.output(saved) == saved and io.output() == saved)
assert(io.write("saved\n") and io.flush())
assert(saved:seek("set") == 0)
io.input(saved)
assert(io.read() == "saved") -- Switching from a granted handle to a native one.
assert(io.close() and io.type(saved) == "closed file")
io.input("AI/config.txt") -- And back again.
assert(io.read() == "hello")
io.close(io.input())

local output = io.output("AI/output.txt")
assert(io.write("output\n") and io.flush())
assert(io.close() and io.type(output) == "closed file")
local outputLines = io.lines("AI/output.txt")
assert(outputLines() == "output" and outputLines() == nil)
local readonly = io.input("AI/config.txt")
io.output(readonly)
assert(io.write("blocked") == nil)
assert(io.flush() == nil)
assert(readonly:setvbuf("no") == nil)
io.close(readonly)
io.input(originalInput)
io.output(originalOutput)
`); err != nil {
				t.Fatal(err)
			}
			if selected {
				// Filename iterators own and close their handle, without changing
				// the default input, even after another input is selected.
				if err := ai.state.DoString(`
local open = io.open
local opened
io.open = function(...) opened = assert(open(...)); return opened end
local default = io.input()
local lines = io.lines("AI/config.txt")
local owned = opened
assert(io.input() == default)
assert(lines() == "hello" and lines() == "world" and lines() == nil)
assert(io.type(owned) == "closed file" and lines() == nil)
local savedLines = io.lines("AI/saved.txt")
owned = opened
assert(savedLines() == "saved" and savedLines() == nil)
assert(io.type(owned) == "closed file")
io.open = open
`); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

type delayedAIAssetFS struct {
	fs.FS
	started chan struct{}
	release chan struct{}
}

func (f *delayedAIAssetFS) ReadFile(name string) ([]byte, error) {
	if name == "shared.txt" {
		close(f.started)
		<-f.release
	}
	return fs.ReadFile(f.FS, name)
}

func TestCompanionAIStateCopyPreservesConcurrentSave(t *testing.T) {
	for _, mode := range []string{"a", "r+"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "shared.txt"), []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			files := &delayedAIAssetFS{FS: os.DirFS(root), started: make(chan struct{}), release: make(chan struct{})}
			resources, err := res.NewManagerFS(files)
			if err != nil {
				t.Fatal(err)
			}
			stateDir := t.TempDir()
			homunculus := &companionAI{stateDir: stateDir}
			mercenary := &companionAI{stateDir: stateDir}
			done := make(chan error, 1)
			go func() {
				_, err := homunculus.prepareIOFile(resources, "shared.txt", mode)
				done <- err
			}()
			defer func() {
				close(files.release)
				select {
				case err := <-done:
					if err != nil {
						t.Error(err)
					}
				case <-time.After(time.Second):
					t.Error("copy did not return")
				}
				got, err := os.ReadFile(filepath.Join(stateDir, "shared.txt"))
				if err != nil || string(got) != "newer saved state" {
					t.Errorf("concurrent update overwritten: got %q, %v", got, err)
				}
			}()
			select {
			case <-files.started:
			case <-time.After(time.Second):
				t.Fatal("copy did not reach asset read")
			}
			file, err := mercenary.prepareIOFile(resources, "shared.txt", "w")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte("newer saved state"), 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

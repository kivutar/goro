//go:build android && cgo

package main

/*
#include <stdlib.h>
char *GoroDocumentList(const char *uri);
int GoroDocumentOpen(const char *uri);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"sync"
	"time"
	"unsafe"
)

type documentEntry struct {
	Filename  string `json:"name"`
	URI       string `json:"uri"`
	Directory bool   `json:"directory"`
	Bytes     int64  `json:"size"`
	Modified  int64  `json:"modified"`
}

func (e documentEntry) Name() string               { return e.Filename }
func (e documentEntry) Size() int64                { return e.Bytes }
func (e documentEntry) ModTime() time.Time         { return time.UnixMilli(e.Modified) }
func (e documentEntry) IsDir() bool                { return e.Directory }
func (e documentEntry) Sys() any                   { return nil }
func (e documentEntry) Type() fs.FileMode          { return e.Mode().Type() }
func (e documentEntry) Info() (fs.FileInfo, error) { return e, nil }
func (e documentEntry) Mode() fs.FileMode {
	if e.Directory {
		return fs.ModeDir | 0555
	}
	return 0444
}

type documentFS struct {
	mu          sync.Mutex
	directories map[string][]fs.DirEntry
	entries     map[string]documentEntry
}

func newDocumentFS(uri string) *documentFS {
	return &documentFS{
		directories: make(map[string][]fs.DirEntry),
		entries:     map[string]documentEntry{".": {Filename: ".", URI: uri, Directory: true}},
	}
}

func (f *documentFS) lookup(name string) (documentEntry, error) {
	if !fs.ValidPath(name) {
		return documentEntry{}, fs.ErrInvalid
	}
	f.mu.Lock()
	entry, ok := f.entries[name]
	f.mu.Unlock()
	if ok {
		return entry, nil
	}
	if _, err := f.ReadDir(path.Dir(name)); err != nil {
		return documentEntry{}, err
	}
	f.mu.Lock()
	entry, ok = f.entries[name]
	f.mu.Unlock()
	if ok {
		return entry, nil
	}
	return documentEntry{}, fs.ErrNotExist
}

func (f *documentFS) Open(name string) (fs.File, error) {
	entry, err := f.lookup(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	if entry.Directory {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fmt.Errorf("is a directory")}
	}
	uri := C.CString(entry.URI)
	defer C.free(unsafe.Pointer(uri))
	fd := int(C.GoroDocumentOpen(uri))
	if fd < 0 {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return os.NewFile(uintptr(fd), name), nil
}

func (f *documentFS) Stat(name string) (fs.FileInfo, error) {
	entry, err := f.lookup(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return entry, nil
}

func (f *documentFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entry, err := f.lookup(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	if !entry.Directory {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fmt.Errorf("not a directory")}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if entries, ok := f.directories[name]; ok {
		return entries, nil
	}
	uri := C.CString(entry.URI)
	defer C.free(unsafe.Pointer(uri))
	raw := C.GoroDocumentList(uri)
	if raw == nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrPermission}
	}
	defer C.free(unsafe.Pointer(raw))
	var documents []documentEntry
	if err := json.Unmarshal([]byte(C.GoString(raw)), &documents); err != nil {
		return nil, err
	}
	entries := make([]fs.DirEntry, 0, len(documents))
	for _, document := range documents {
		if document.Filename != "." && fs.ValidPath(document.Filename) && path.Base(document.Filename) == document.Filename {
			entries = append(entries, document)
			f.entries[path.Join(name, document.Filename)] = document
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	f.directories[name] = entries
	return entries, nil
}

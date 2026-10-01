package game

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/res"
	lua "github.com/yuin/gopher-lua"
)

// Homunculus and mercenary scripts share saved files. Readers and openers
// must wait for initial copies to finish; script writes keep normal file IO.
var aiStateFilesMu sync.Mutex

func (ai *companionAI) statePath(name string) (string, error) {
	name = path.Clean(strings.ReplaceAll(strings.Trim(strings.TrimSpace(name), `"'`), "\\", "/"))
	if name == "." || !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	return filepath.Join(ai.stateDir, filepath.FromSlash(name)), nil
}

// Both dofile/require and io.open read saved state before the selected assets.
func (ai *companionAI) readAIFile(resources *res.Manager, name string) ([]byte, error) {
	if ai.stateDir != "" {
		filename, err := ai.statePath(name)
		if err != nil {
			return nil, err
		}
		aiStateFilesMu.Lock()
		data, err := os.ReadFile(filename)
		aiStateFilesMu.Unlock()
		if !os.IsNotExist(err) {
			return data, err
		}
	}
	return resources.ReadFile(name)
}

// Empty means a read-only asset: Lua receives an in-memory reader, without
// copying the file to disk. Modified files live in the per-folder state dir.
func (ai *companionAI) prepareIOFile(resources *res.Manager, name, mode string) (string, error) {
	filename, err := ai.statePath(name)
	if err != nil {
		return "", err
	}
	var write, preserve, requireExisting bool
	switch mode {
	case "", "r", "rb":
	case "w", "wb", "w+", "wb+":
		write = true
	case "a", "ab", "a+", "ab+":
		write, preserve = true, true
	case "r+", "rb+":
		write, preserve, requireExisting = true, true, true
	default:
		return "", fmt.Errorf("invalid file mode %q", mode)
	}
	aiStateFilesMu.Lock()
	_, err = os.Stat(filename)
	aiStateFilesMu.Unlock()
	if !os.IsNotExist(err) {
		return filename, err
	}
	if !write {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", err
	}
	var data []byte
	if preserve {
		if resources.HasFileExact(name) {
			data, err = resources.ReadFileExact(name)
			if err != nil {
				return "", err
			}
		} else if requireExisting {
			return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	if err := createAIStateFile(filename, data); err != nil {
		return "", err
	}
	return filename, nil
}

func createAIStateFile(filename string, data []byte) error {
	aiStateFilesMu.Lock()
	defer aiStateFilesMu.Unlock()
	// Another AI may have saved this file while the original was being read.
	// Exclusive creation preserves that file, including for append/r+ opens.
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		// Never leave a partial override masking the original asset.
		_ = os.Remove(filename)
	}
	return err
}

func companionAIIOError(L *lua.LState, err error) int {
	L.Push(lua.LNil)
	L.Push(lua.LString(err.Error()))
	return 2
}

// GopherLua's native file userdata only accepts os.File opened by its own
// io.open. This read-only handle provides seek/read/lines for granted assets;
// writable state continues to use the native Lua file implementation.
type aiReadFile struct {
	data   *bytes.Reader
	reader *bufio.Reader
	closed bool
}

func pushAIReadFile(L *lua.LState, data []byte) {
	f := &aiReadFile{data: bytes.NewReader(data)}
	f.reader = bufio.NewReader(f.data)
	u := L.NewUserData()
	u.Value = f
	mt := L.NewTypeMetatable("goro.ai.readfile")
	L.SetFuncs(mt, map[string]lua.LGFunction{
		"read":  aiFileRead,
		"seek":  aiFileSeek,
		"close": aiFileClose,
		"lines": func(L *lua.LState) int {
			u := L.CheckUserData(1)
			file := checkAIReadFile(L)
			L.Push(L.NewFunction(func(L *lua.LState) int {
				L.SetTop(0)
				L.Push(u)
				if file.closed {
					L.RaiseError("file is closed")
				}
				return aiFileRead(L)
			}))
			return 1
		},
		"write":   aiFileReadOnly,
		"flush":   aiFileReadOnly,
		"setvbuf": aiFileReadOnly,
	})
	mt.RawSetString("__index", mt)
	L.SetMetatable(u, mt)
	L.Push(u)
}

func aiFileReadOnly(L *lua.LState) int {
	checkAIReadFile(L)
	return companionAIIOError(L, fmt.Errorf("file is opened for only reading"))
}

func checkAIReadFile(L *lua.LState) *aiReadFile {
	f, ok := L.CheckUserData(1).Value.(*aiReadFile)
	if !ok {
		L.ArgError(1, "file expected")
	}
	if f.closed {
		L.RaiseError("file is closed")
	}
	return f
}

func aiFileClose(L *lua.LState) int {
	f := checkAIReadFile(L)
	f.closed, f.data, f.reader = true, nil, nil
	L.Push(lua.LTrue)
	return 1
}

func aiFileSeek(L *lua.LState) int {
	f := checkAIReadFile(L)
	whence := L.OptString(2, "cur")
	offset := L.OptInt64(3, 0)
	base := io.SeekStart
	switch whence {
	case "set":
	case "cur":
		base = io.SeekCurrent
		offset -= int64(f.reader.Buffered())
	case "end":
		base = io.SeekEnd
	default:
		L.ArgError(2, "invalid seek mode")
	}
	pos, err := f.data.Seek(offset, base)
	if err != nil {
		return companionAIIOError(L, err)
	}
	f.reader.Reset(f.data)
	L.Push(lua.LNumber(pos))
	return 1
}

func aiFileRead(L *lua.LState) int {
	f := checkAIReadFile(L)
	if L.GetTop() == 1 {
		L.Push(lua.LString("*l"))
	}
	top := L.GetTop()
	for i := 2; i <= top; i++ {
		var data []byte
		var err error
		if count, ok := L.Get(i).(lua.LNumber); ok {
			if count < 0 {
				L.ArgError(i, "invalid byte count")
			}
			available := f.reader.Buffered() + f.data.Len()
			if available == 0 {
				err = io.EOF
			} else {
				data = make([]byte, int(min(float64(count), float64(available))))
				_, err = io.ReadFull(f.reader, data)
			}
		} else {
			switch L.CheckString(i) {
			case "*a":
				data, err = io.ReadAll(f.reader)
			case "*l":
				data, err = f.reader.ReadBytes('\n')
				if len(data) > 0 {
					if data[len(data)-1] == '\n' {
						data = bytes.TrimSuffix(data[:len(data)-1], []byte{'\r'})
					}
					err = nil
				}
			case "*n":
				var number float64
				_, err = fmt.Fscanf(f.reader, "%f", &number)
				if err == nil {
					L.Push(lua.LNumber(number))
					continue
				}
			default:
				L.ArgError(i, "invalid read format")
			}
		}
		if err == io.EOF {
			L.Push(lua.LNil)
			break
		}
		if err != nil {
			return companionAIIOError(L, err)
		}
		L.Push(lua.LString(data))
	}
	return L.GetTop() - top
}

func registerAIReadFileIO(L *lua.LState, ioTable *lua.LTable) {
	for _, name := range []string{"close", "type"} {
		original := ioTable.RawGetString(name)
		ioTable.RawSetString(name, L.NewFunction(func(L *lua.LState) int {
			if u, ok := L.Get(1).(*lua.LUserData); ok {
				if file, ok := u.Value.(*aiReadFile); ok {
					if name == "close" {
						return aiFileClose(L)
					}
					kind := "file"
					if file.closed {
						kind = "closed file"
					}
					L.Push(lua.LString(kind))
					return 1
				}
			}
			top := L.GetTop()
			L.Push(original)
			for i := 1; i <= top; i++ {
				L.Push(L.Get(i))
			}
			L.Call(top, lua.MultRet)
			return L.GetTop() - top
		}))
	}
	// Dispatch through file methods so both native files and granted assets
	// work as default streams. Filename forms must use our io.open as well.
	const source = `
local input, output = io.input(), io.output()
local close = io.close
local function selectFile(value, current, mode)
	if value == nil then return current end
	if type(value) == "string" then
		local file, err = io.open(value, mode)
		if not file then error(err, 3) end
		return file
	end
	if io.type(value) ~= "file" then error("expected an open file", 3) end
	return value
end
io.input = function(value)
	input = selectFile(value, input, "r")
	return input
end
io.output = function(value)
	output = selectFile(value, output, "w")
	return output
end
io.read = function(...) return input:read(...) end
io.write = function(...) return output:write(...) end
io.flush = function() return output:flush() end
io.close = function(file) return close(file or output) end
io.lines = function(filename)
	local file = selectFile(filename, input, "r")
	local owned = filename ~= nil
	local done = false
	return function()
		if done then return nil end
		local line, err = file:read()
		if err then error(err, 2) end
		if line == nil and owned then
			close(file)
			done = true
		end
		return line
	end
end
`
	if err := L.DoString(source); err != nil {
		glog.Warnf("AI file IO setup failed: %v", err)
	}
}

// Package atomicfile replaces files without ever leaving them half written.
// cuck edits files it doesn't own (shell startup files, PowerShell profiles),
// so a full disk or a killed process must not cost the user their contents.
package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Write replaces path with data. It writes a temporary file next to it and
// renames it into place, so path holds either its old contents or data,
// never a mix. A symlink at path is followed, so dotfile managers keep their
// links. When the directory isn't writable but the file is, it falls back to
// writing the file in place.
func Write(path string, data []byte, perm fs.FileMode) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".cuck-*")
	if errors.Is(err, fs.ErrPermission) {
		return os.WriteFile(path, data, perm)
	}
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(name, perm)
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		os.Remove(name)
	}
	return err
}

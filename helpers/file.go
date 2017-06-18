package helpers

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

// VisitCreatedFile calls os.Create to create a file, and applies w to it.
func VisitCreatedFile(name string, w func(io.Writer) error) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			_ = f.Close() // nolint: gas
		}
	}()
	if err := w(f); err != nil {
		return err
	}
	close = false
	return f.Close()
}

// CopyFileContents copies the file contents from src to dst.
// It's not atomic and doesn't copy permissions or metadata.
func CopyFileContents(dst, src string, perm os.FileMode) error {
	// nolint: gas
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() // nolint: errcheck, gas
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		_ = os.Remove(dst) // nolint: gas
		return err
	}
	return out.Close()
}

// ReadFileMagic returns the first four bytes of the file, with final '\r' replaced by '\n'.
func ReadFileMagic(name string) (data []byte, err error) {
	f, err := os.Open(name)
	if err != nil {
		return
	}
	defer f.Close() // nolint: errcheck
	data = make([]byte, 4)
	_, err = f.Read(data)
	if err == io.EOF {
		err = nil
	}
	// Normalize windows linefeeds. This function is used to
	// recognize frontmatter, so we only need to look at the fourth byte.
	if data[3] == '\r' {
		data[3] = '\n'
	}
	return
}

// PostfixWalk is like filepath.Walk, but visits each directory after visiting its children instead of before.
// It does not implement SkipDir.
func PostfixWalk(root string, walkFn filepath.WalkFunc) error {
	if files, err := ioutil.ReadDir(root); err == nil {
		for _, stat := range files {
			if stat.IsDir() {
				if err = PostfixWalk(filepath.Join(root, stat.Name()), walkFn); err != nil {
					return err
				}
			}
		}
	}

	info, err := os.Stat(root)
	return walkFn(root, info, err)
}

// IsNotEmpty returns a boolean indicating whether the error is known to report that a directory is not empty.
func IsNotEmpty(err error) bool {
	if err, ok := err.(*os.PathError); ok {
		return err.Err.(syscall.Errno) == syscall.ENOTEMPTY
	}
	return false
}

// NewPathError returns an os.PathError that formats as the given text.
func NewPathError(op, name, text string) *os.PathError {
	return &os.PathError{Op: op, Path: name, Err: errors.New(text)}
}

// PathError returns an instance of *os.PathError, by wrapping its argument
// if it is not already an instance.
// PathError returns nil for a nil argument.
func PathError(err error, op, name string) *os.PathError {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *os.PathError:
		return err
	default:
		return &os.PathError{Op: "read", Path: name, Err: err}
	}
}

// RemoveEmptyDirectories recursively removes empty directories.
func RemoveEmptyDirectories(root string) error {
	walkFn := func(name string, info os.FileInfo, err error) error {
		switch {
		case err != nil && os.IsNotExist(err):
			// It's okay to call this on a directory that doesn't exist.
			// It's also okay if another process removed a file during traversal.
			return nil
		case err != nil:
			return err
		case info.IsDir():
			err := os.Remove(name)
			switch {
			case err == nil:
				return nil
			case os.IsNotExist(err):
				return nil
			case IsNotEmpty(err):
				return nil
			default:
				return err
			}
		}
		return nil
	}
	return PostfixWalk(root, walkFn)
}
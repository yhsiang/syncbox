package syncbox

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

//go:generate callbackgen -type FileWatcher
type FileWatcher struct {
	path  string
	ctx   context.Context
	files map[string]File

	changeCallbacks []func(files []File)
}

type File struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	State    string
	// isDirectory bool
	// Size        int64
}

func (f *File) CalChecksum() error {
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	f.Checksum = hex.EncodeToString(hash.Sum(nil))

	return nil
}

func (f *FileWatcher) WalkDir() error {
	var changes []File
	var newFiles = make(map[string]File)
	err := filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("test %+v", err)
		}

		if !info.IsDir() {
			file := File{
				Name: info.Name(),
				Path: path,
			}

			err2 := file.CalChecksum()
			if err2 != nil {
				return err2
			}

			newFiles[file.Name] = file
		}

		return nil
	})

	for _, oldFile := range f.files {
		newFile, ok := newFiles[oldFile.Name]
		if !ok {
			oldFile.State = "delete"
			changes = append(changes, oldFile)
		}

		if ok && oldFile.Checksum != newFile.Checksum {
			oldFile.State = "update"
			changes = append(changes, oldFile)
		}
	}

	for _, newFile := range newFiles {
		if _, ok := f.files[newFile.Name]; !ok {
			newFile.State = "new"
			changes = append(changes, newFile)
		}
	}

	f.files = newFiles

	if len(changes) > 0 {
		f.EmitChange(changes)
	}

	return err
}

func (f *FileWatcher) Run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	err := f.WalkDir()
	if err != nil {
		// log error
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			err := f.WalkDir()
			if err != nil {
				// log error
			}
		}
	}
}

func NewFileWatcher(ctx context.Context, path string) *FileWatcher {
	return &FileWatcher{
		path:  path,
		ctx:   ctx,
		files: make(map[string]File),
	}
}

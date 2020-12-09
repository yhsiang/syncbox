package syncbox

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/google/uuid"
)

type ID string

//go:generate callbackgen -type FileWatcher
type FileWatcher struct {
	mu              sync.Mutex
	path            string
	ctx             context.Context
	files           map[string]File
	downloads       map[ID]File
	changeCallbacks []func(files []File)
}

type File struct {
	Name     string    `json:"name"`
	RootPath string    `json:"-"`
	Path     string    `json:"path"`
	Checksum string    `json:"checksum"`
	State    string    `json:"-"`
	Action   string    `json:"action"`
	Content  io.Reader `json:"-"`
	ID       ID        `json:"id"`
}

func (f *File) CalChecksum() error {
	file, err := os.Open(fmt.Sprintf("%s%s", f.RootPath, f.FullName()))
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

func (f *File) FullName() string {
	return fmt.Sprintf("%s%s", f.Path, f.Name)
}

type FileSlice []File

func (files FileSlice) toMap() map[string]File {
	var maps = make(map[string]File)
	for _, file := range files {
		maps[file.FullName()] = file
	}

	return maps
}

func (f *FileWatcher) WalkDir() error {
	var changes []File
	var newFiles = make(map[string]File)
	err := filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		if err != nil {
			log.WithError(err).Error("walk error")
		}

		if !info.IsDir() {
			var fileName = info.Name()
			var pathFileName = strings.Replace(path, f.path, "", 1)
			var pathOnly = strings.Replace(pathFileName, fileName, "", 1)
			file := File{
				Name:     fileName,
				RootPath: f.path,
				Path:     pathOnly,
				ID:       ID(uuid.New().String()),
			}

			err2 := file.CalChecksum()
			if err2 != nil {
				return err2
			}

			log.Debugf("%+v", file)
			newFiles[file.FullName()] = file
		}

		return nil
	})

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, oldFile := range f.files {
		newFile, ok := newFiles[oldFile.FullName()]
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
		if _, ok := f.files[newFile.FullName()]; !ok {
			newFile.State = "new"
			changes = append(changes, newFile)
		}

		f.downloads[newFile.ID] = newFile
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
		log.WithError(err).Error("run error")
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			err := f.WalkDir()
			if err != nil {
				log.WithError(err).Error("run error")
			}
		}
	}
}

func (f *FileWatcher) Compare(files FileSlice) FileSlice {
	f.mu.Lock()
	defer f.mu.Unlock()

	var syncingFiles = FileSlice{} // avoid null
	for _, file := range files {
		if _, ok := f.files[file.FullName()]; !ok {
			file.Action = "upload"
			syncingFiles = append(syncingFiles, file)
		}
	}

	var maps = files.toMap()
	log.Infof("maps %+v", maps)
	for key, file := range f.files {
		if _, ok := maps[key]; !ok {
			log.Infof("file %+v", file)
			file.Action = "download"
			syncingFiles = append(syncingFiles, file)
		}
	}

	return syncingFiles
}

func (f *FileWatcher) Set(file File) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.files[file.FullName()] = file
}

func NewFileWatcher(ctx context.Context, path string) *FileWatcher {
	return &FileWatcher{
		path:      path,
		ctx:       ctx,
		files:     make(map[string]File),
		downloads: make(map[ID]File),
	}
}

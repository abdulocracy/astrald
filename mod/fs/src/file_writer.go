package fs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/cryptopunkscc/astrald/data"
	"github.com/cryptopunkscc/astrald/mod/fs"
	"os"
	"path/filepath"
	"sync/atomic"
)

type FileWriter struct {
	path      string
	tempID    string
	file      *os.File
	resolver  data.Resolver
	store     *StoreService
	finalized atomic.Bool
}

func NewFileWriter(parent *StoreService, path string) (*FileWriter, error) {
	var rbytes = make([]byte, 8)
	rand.Read(rbytes)

	var tempID = ".tmp." + hex.EncodeToString(rbytes)

	file, err := os.Create(filepath.Join(path, tempID))
	if err != nil {
		return nil, err
	}

	resolver := data.NewResolver()

	return &FileWriter{
		path:     path,
		tempID:   tempID,
		file:     file,
		resolver: resolver,
		store:    parent,
	}, nil
}

func (w *FileWriter) Write(p []byte) (n int, err error) {
	n, err = w.file.Write(p)

	if n > 0 {
		w.resolver.Write(p[:n])
	}

	return n, err
}

func (w *FileWriter) Commit() (data.ID, error) {
	if !w.finalized.CompareAndSwap(false, true) {
		return data.ID{}, errors.New("writer closed")
	}

	w.file.Close()

	dataID := w.resolver.Resolve()

	var oldPath = filepath.Join(w.path, w.tempID)
	var newPath = filepath.Join(w.path, dataID.String())

	err := os.Rename(oldPath, newPath)
	if err != nil {
		os.Remove(oldPath)
	}

	if w.store != nil {
		w.store.index.AddToSet(nameReadWrite, dataID)
		w.store.events.Emit(fs.EventFileAdded{
			DataID: dataID,
			Path:   newPath,
		})
	}

	return dataID, err
}

func (w *FileWriter) Discard() error {
	if !w.finalized.CompareAndSwap(false, true) {
		return errors.New("writer closed")
	}

	w.file.Close()
	os.Remove(filepath.Join(w.path, w.tempID))
	return nil
}

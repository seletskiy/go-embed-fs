// Package embedfs provides a way to embed files and directories into
// the binary blob, such as executable file.
//
// Using embedfs is possible to do such silly (and wonderful!) things like
// embedding git repository directly into executable binary which operates on.
//
// For obvious reasons embedfs is read-only filesystem.
package embedfs

import (
	"archive/tar"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNotAvail       = errors.New("not available, embedfs is read only file system")
	ErrNoExist        = errors.New("file is not exist")
	ErrNoFootprint    = errors.New("no embedfs footprint found")
	ErrInvalidOffset  = errors.New("embedfs offset is out of bounds of file")
	ErrNotImplemented = errors.New("not implemented yet")
)

const signatureLen = 12

var (
	signature = [signatureLen]byte{
		'E', 'M', 'B', 'E', 'D', 'F', 'S', '~', '0', '0', '0', ':',
	}
)

// EmbedFs represents read-only instance of embedded fs, which can be used
// for accessing previously embedded files and directories.
type EmbedFs struct {
	files  []*embedFsEntry
	index  map[string]*embedFsEntry
	origin file
	offset int64
}

type embedFsEntry struct {
	name   string
	offset int64
	header *tar.Header
}

type embedFsFootprint struct {
	Signature [signatureLen]byte
	Offset    int64
}

type Embedder struct {
	writer *tar.Writer
	offset int64
	origin file
}

type embedFileReader struct {
	name   string
	start  int64
	length int64
	offset int64
	source file
}

type file interface {
	io.Closer
	io.Writer
	io.Reader
	io.ReaderAt
	io.Seeker
	Stat() (os.FileInfo, error)
	Truncate(size int64) error
}

// Open will return embedfs if it's available in specified source file.
//
// That embedfs should first be created by method Create.
//
// It will accept common file as it's argument, os.File will server well.
func Open(origin file) (*EmbedFs, error) {
	stat, err := origin.Stat()
	if err != nil {
		return nil, err
	}

	footprint := embedFsFootprint{}
	_, err = origin.Seek(-int64(binary.Size(footprint)), os.SEEK_END)
	if err != nil {
		return nil, err
	}

	err = binary.Read(origin, binary.BigEndian, &footprint)
	if err != nil {
		return nil, err
	}

	if footprint.Signature != signature {
		return nil, ErrNoFootprint
	}

	if footprint.Offset >= stat.Size() || footprint.Offset < 0 {
		return nil, ErrInvalidOffset
	}

	fs := &EmbedFs{
		files:  []*embedFsEntry{},
		index:  map[string]*embedFsEntry{},
		origin: origin,
		offset: footprint.Offset,
	}

	_, err = origin.Seek(fs.offset, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(origin)

	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fs, err
		}

		seek, _ := origin.Seek(0, os.SEEK_CUR)
		entry := &embedFsEntry{
			name:   tarHeader.Name,
			offset: seek,
			header: tarHeader,
		}

		fs.files = append(fs.files, entry)
		fs.index[entry.name] = entry
	}

	return fs, nil
}

// Truncate erases all embedfs data from the specified file, leaving it
// in the state it was before embedding has been done.
func Truncate(origin file) error {
	fs, err := Open(origin)
	if err != nil {
		return err
	}

	return origin.Truncate(fs.offset)
}

// Create creates new embedfs in the end of specified file.
//
// It will return Embedder, which can be used for storing files and directories
// in that embedfs.
//
// After all files were added, Close method should be invoked to correctly
// finish embedfs data.
func Create(origin file) (*Embedder, error) {
	currentSeek, err := origin.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	return &Embedder{
		writer: tar.NewWriter(origin),
		offset: currentSeek,
		origin: origin,
	}, nil
}

// EmbedFile used for embedding single file to the embedded fs.
//
// Specified file will be added to the end of list.
func (e Embedder) EmbedFile(path string, target string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	tarHeader, err := tar.FileInfoHeader(stat, "")
	if err != nil {
		return err
	}

	tarHeader.Name = filepath.Join("/", target)
	e.writer.WriteHeader(tarHeader)
	if err != nil {
		return err
	}

	sourceFile, err := os.Open(path)
	if err != nil {
		return err
	}

	defer sourceFile.Close()

	_, err = io.Copy(e.writer, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

// EmbedDirectory used for embedding entire directory to the embedded fs.
//
// It's simple wrapper under filepath.Walk and EmbedFile.
func (e Embedder) EmbedDirectory(root, prefix string) error {
	return filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			return e.EmbedFile(path,
				filepath.Join(prefix, strings.TrimPrefix(path, root)))
		},
	)
}

// Close stops embedding process and write end marker to the container file.
//
// After this invokation embedded fs are no longer write-capable.
func (e Embedder) Close() error {
	err := e.writer.Close()
	if err != nil {
		return err
	}

	err = binary.Write(e.origin, binary.BigEndian, embedFsFootprint{
		signature,
		e.offset,
	})

	return err
}

// Open opens specified file from embedded fs for reading only.
func (fs *EmbedFs) Open(path string) (file, error) {
	path = filepath.Join("/", path)

	if !fs.IsFileExist(path) {
		return nil, ErrNoExist
	}

	return &embedFileReader{
		start:  fs.index[path].offset,
		length: fs.index[path].header.Size,
		source: fs.origin,
		name:   path,
	}, nil
}

// ListDir return list of files in embedded fs in the order they was added.
func (fs EmbedFs) ListDir(path string) ([]string, error) {
	result := []string{}

	for _, entry := range fs.files {
		rootName := filepath.Join("/", entry.name)
		if strings.HasPrefix(rootName, filepath.Join(path, "/")) {
			result = append(result, entry.name)
		}
	}

	return result, nil
}

// IsFileExist return true, if specified file exist in embedded fs.
func (fs *EmbedFs) IsFileExist(path string) bool {
	_, exist := fs.index[path]
	return exist
}

// Create operation does not supported. For interface compatibility only.
func (fs *EmbedFs) Create(path string) (file, error) {
	return nil, ErrNotAvail
}

// Create operation does not supported. For interface compatibility only.
func (fs EmbedFs) TempFile() (file, error) {
	return nil, ErrNotAvail
}

// Create operation does not supported. For interface compatibility only.
func (fs *EmbedFs) Move(from string, to string) error {
	return ErrNotAvail
}

// Close closes previously opened file. For interface compatibility only.
func (fs *EmbedFs) Close() error {
	return fs.origin.Close()
}

// Read is standard read funciton implementation from io.Reader.
func (reader *embedFileReader) Read(b []byte) (int, error) {
	rest := reader.length - reader.offset
	if rest <= 0 {
		return 0, io.EOF
	}

	n, err := reader.source.ReadAt(b, reader.start+reader.offset)

	if rest < int64(n) {
		reader.offset += int64(rest)
		return int(rest), err
	} else {
		reader.offset += int64(n)
		return n, err
	}
}

// Write operation is not supported. For interface compatibility only.
func (reader *embedFileReader) Write(b []byte) (int, error) {
	return 0, ErrNotAvail
}

// Name returns name of the embedded file.
func (reader *embedFileReader) Name() string {
	return reader.name
}

// Close closes previously opened file. For interface compatibility only.
func (reader *embedFileReader) Close() error {
	return reader.source.Close()
}

// ReadAt operation is not implemeted yet.
func (reader *embedFileReader) ReadAt(p []byte, off int64) (int, error) {
	return 0, ErrNotImplemented
}

// Seek operation is not implemeted yet.
func (reader *embedFileReader) Seek(offset int64, whence int) (int64, error) {
	return 0, ErrNotImplemented
}

// Stat operation is not implemeted yet.
func (reader *embedFileReader) Stat() (os.FileInfo, error) {
	return nil, ErrNotImplemented
}

// Truncate operation is not supported. For interface compatibility only.
func (reader *embedFileReader) Truncate(int64) error {
	return ErrNotAvail
}

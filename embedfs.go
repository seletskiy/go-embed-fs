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

type embedFs struct {
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

type embedder struct {
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
}

func OpenEmbedFs(origin file) (*embedFs, error) {
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

	fs := &embedFs{
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

func CreateEmbedFs(origin file) (*embedder, error) {
	currentSeek, err := origin.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	return &embedder{
		writer: tar.NewWriter(origin),
		offset: currentSeek,
		origin: origin,
	}, nil
}

func (e embedder) EmbedFile(path string, target string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	tarHeader, err := tar.FileInfoHeader(stat, "")
	if err != nil {
		return err
	}

	tarHeader.Name = target
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

func (e embedder) EmbedDirectory(root string) error {
	return filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			return e.EmbedFile(path, strings.TrimPrefix(path, root))
		},
	)
}

func (e embedder) Close() error {
	err := e.writer.Close()
	if err != nil {
		return err
	}

	_, err = e.origin.Write(signature[:])
	if err != nil {
		return err
	}

	offsetBuf := make([]byte, binary.Size(e.offset))
	binary.PutVarint(offsetBuf, e.offset)

	_, err = e.origin.Write(offsetBuf)
	if err != nil {
		return err
	}

	return nil
}

func (fs *embedFs) Open(path string) (file, error) {
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

func (fs embedFs) ListDir(path string) ([]string, error) {
	// @TODO
	return nil, ErrNotImplemented
}

func (fs *embedFs) IsFileExist(path string) bool {
	_, exist := fs.index[path]
	return exist
}

func (fs *embedFs) Create(path string) (file, error) {
	return nil, ErrNotAvail
}

func (fs embedFs) TempFile() (file, error) {
	return nil, ErrNotAvail
}

func (fs *embedFs) Move(from string, to string) error {
	return ErrNotAvail
}

func (fs *embedFs) Close() error {
	return fs.origin.Close()
}

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

func (reader *embedFileReader) Write(b []byte) (int, error) {
	return 0, ErrNotAvail
}

func (reader *embedFileReader) Name() string {
	return reader.name
}

func (reader *embedFileReader) Close() error {
	return reader.source.Close()
}

func (reader *embedFileReader) ReadAt(p []byte, off int64) (int, error) {
	return 0, ErrNotImplemented
}

func (reader *embedFileReader) Seek(offset int64, whence int) (int64, error) {
	return 0, ErrNotImplemented
}

func (reader *embedFileReader) Stat() (os.FileInfo, error) {
	return nil, ErrNotImplemented
}

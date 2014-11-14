package embedfs

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/seletskiy/go-mock-file"
)

func TestCanCreateEmptyFs(t *testing.T) {
	container := mockfile.New("la")

	embedder, err := Create(container)
	if err != nil {
		panic(err)
	}

	err = embedder.Close()
	if err != nil {
		panic(err)
	}

	_, err = Open(container)
	if err != nil {
		panic(err)
	}
}

func TestCanEmbedSingleFile(t *testing.T) {
	container := mockfile.New("lala")

	embedder, err := Create(container)

	err = embedder.EmbedFile("embedfs.go", "embedfs.go")
	if err != nil {
		panic(err)
	}

	err = embedder.Close()
	if err != nil {
		panic(err)
	}

	fs, err := Open(container)
	if err != nil {
		panic(err)
	}

	if !fs.IsFileExist("embedfs.go") {
		t.Fatal("file <embedfs.go> is not exist in embedfs")
	}
}

func TestCanEmbedDirectory(t *testing.T) {
	container := mockfile.New("lala3")

	embedder, err := Create(container)
	if err != nil {
		panic(err)
	}

	err = embedder.EmbedDirectory(".")
	if err != nil {
		panic(err)
	}

	err = embedder.Close()
	if err != nil {
		panic(err)
	}

	fs, err := Open(container)
	if err != nil {
		panic(err)
	}

	if !fs.IsFileExist("embedfs.go") {
		t.Fatal("file <embedfs.go> is not exist in embedfs")
	}

	if !fs.IsFileExist("embedfs_test.go") {
		t.Fatal("file <embedfs_test.go> is not exist in embedfs")
	}
}

func TestCanReadFile(t *testing.T) {
	container := mockfile.New("lala3")

	embedder, err := Create(container)
	if err != nil {
		panic(err)
	}

	err = embedder.EmbedFile("embedfs.go", "embedfs.go")
	if err != nil {
		panic(err)
	}

	err = embedder.Close()
	if err != nil {
		panic(err)
	}

	fs, err := Open(container)
	if err != nil {
		panic(err)
	}

	expected, err := ioutil.ReadFile("embedfs.go")
	if err != nil {
		panic(err)
	}

	f, err := fs.Open("embedfs.go")
	if err != nil {
		panic(err)
	}

	actual, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("file from embedfs is not equal to actual file")
	}
}

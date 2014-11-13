package embedfs

import (
	"testing"

	"github.com/seletskiy/go-mock-file"
)

func TestCanCreateEmptyFs(t *testing.T) {
	file := mockfile.New("la")

	embedder, err := CreateEmbedFs(file)
	if err != nil {
		panic(err)
	}

	err = embedder.Close()
	if err != nil {
		panic(err)
	}

	fs, err := OpenEmbedFs(file)
	if err != nil {
		panic(err)
	}
}

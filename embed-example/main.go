package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/seletskiy/go-embed-fs"
)

func main() {
	usage := `EmbedFS Example embedding.

Usage:
  embed-example -h | --help
  embed-example -I
  embed-example -E <target> <file>...
  embed-example -C <file>
  embed-example -L
  embed-example -T <target>

Options:
  -h --help  Show this screen.
  -I         Check that current binary contains embedfs.
  -E         Embed specified <file>s into <target> binary.
  -C         Print contents of specified file to stdout.
  -L         List embedded files.
  -T         Truncate current binary and write clean binary to <target>.`

	args, _ := docopt.Parse(usage, nil, true, "EmbedFS Example", false)

	switch {
	case args["-E"]:
		EmbedFiles(
			os.Args[0],
			args["<target>"].(string),
			args["<file>"].([]string),
		)
	case args["-L"]:
		ListFiles(os.Args[0])
	case args["-C"]:
		CatFile(os.Args[0], args["<file>"].([]string)[0])
	case args["-T"]:
		Truncate(os.Args[0], args["<target>"].(string))
	case args["-I"]:
		Check(os.Args[0])
	}
}

func EmbedFiles(sourceName string, embedFsFileName string, files []string) {
	target, err := os.Create(embedFsFileName)
	if err != nil {
		log.Fatalf(`can't open <%s> for writing: %s`, embedFsFileName, err)
	}

	err = os.Chmod(embedFsFileName, 0700)
	if err != nil {
		log.Fatalf(`can't chmod <%s> to 0700: %s`, embedFsFileName, err)
	}

	source, err := os.Open(sourceName)
	if err != nil {
		log.Fatalf(`can't open <%s> for reading: %s`, source, err)
	}

	io.Copy(target, source)

	embedder, err := embedfs.Create(target)
	if err != nil {
		log.Fatalf(`can't create embedfs on <%s>: %s`, embedFsFileName, err)
	}

	defer embedder.Close()

	for _, fileName := range files {
		err := embedder.EmbedFile(fileName, fileName)
		if err != nil {
			log.Printf(`can't embed file <%s> into <%s>: %s`,
				fileName,
				embedFsFileName,
				err.Error(),
			)
		}
	}
}

func ListFiles(embedFsFileName string) {
	fs, err := openEmbedFs(embedFsFileName)
	if err != nil {
		log.Fatalf(`can't open embedfs: %s`, err)
	}

	contents, _ := fs.ListDir("/")
	for _, entry := range contents {
		fmt.Println(entry)
	}
}

func CatFile(embedFsFileName string, fileName string) {
	fs, err := openEmbedFs(embedFsFileName)
	if err != nil {
		log.Fatalf(`can't open embedfs: %s`, err)
	}

	file, err := fs.Open(fileName)
	if err != nil {
		log.Fatalf(`can't open file <%s> in embedfs: %s`, fileName, err)
	}

	io.Copy(os.Stdout, file)
}

func Truncate(embedFsFileName string, targetName string) {
	target, err := os.Create(targetName)
	if err != nil {
		log.Fatalf(`can't open <%s> for writing: %s`, targetName, err)
	}

	defer target.Close()

	err = os.Chmod(targetName, 0700)
	if err != nil {
		log.Fatalf(`can't chmod <%s> to 0700: %s`, targetName, err)
	}

	source, err := os.Open(embedFsFileName)
	if err != nil {
		log.Fatalf(`can't open <%s> for reading: %s`, embedFsFileName, err)
	}

	io.Copy(target, source)

	err = embedfs.Truncate(target)
	if err != nil {
		log.Fatalf(`can't truncate embedfs: %s`, err)
	}
}

func Check(embedFsFileName string) {
	_, err := openEmbedFs(embedFsFileName)

	if err != nil {
		fmt.Printf(
			"<%s> doesn't contain embedded fs.\n",
			embedFsFileName,
		)
	} else {
		fmt.Printf(
			"<%s> contains embedded fs; use -L to list files.\n",
			embedFsFileName,
		)
	}
}

func openEmbedFs(sourceName string) (*embedfs.EmbedFs, error) {
	source, err := os.Open(sourceName)
	if err != nil {
		return nil, err
	}

	fs, err := embedfs.Open(source)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

Go Embedded FS
==============

This packages provides a convient way to embed files and directories directly
into executable binary (or any else binary).

Documentation: http://godoc.org/github.com/seletskiy/go-embed-fs

Usage
=====

Embedded FS is read-only file system, so you need to create it in order to be
used.

```
containerFile, err := os.OpenFile(containerFileName, os.O_RDWR, 0755)
// check for err

embedder, err := embedfs.Create(targetFile)
// check for err

embedder.EmbedFile(sourceFileName, targetFileName)
embedder.EmbedDirectory(sourceDirName, targetDirName)
// ... any number of times

// this step is NECESSARY!
embedder.Close()

fs, err := embedfs.Open(containerFileName)
// check for err

// embedded fs is now ready to be used for reading files:
file, err := fs.Open(someFileName)
files, err := fs.ListDir(someDirName)

```

Example
=======

See `embed-example/main.go` in this repo.

Use `cd embed-example/ && go build` to build play tool for embedfs.

`./embed-example` binary should appear after build.

Example session:

```
$ ./embed-example -I
<./embed-example> doesn't contain embedded fs.

$ ./embed-example -E binary-with-data main.go

$ ./binary-with-data -I
<./binary-with-data> contains embedded fs; use -L to list files.

$ ./binary-with-data -L
/main.go

$ md5sum main.go
091adaf415dd55a9db4e2d3d574b3665  main.go

$ ./binary-with-data -C main.go | md5sum
091adaf415dd55a9db4e2d3d574b3665  -

$ md5sum ./embed-example
9e74d9899d745d0d97386cb56bdec449  ./embed-example

$ md5sum ./binary-with-data
a880841e54d5eb6842af4712ff85d1b9  ./binary-with-data

$ ./binary-with-data -T binary-without-data

$ md5sum ./binary-without-data
9e74d9899d745d0d97386cb56bdec449  ./binary-without-data

$ ./binary-without-data -I
<./binary-without-data> doesn't contain embedded fs.
```

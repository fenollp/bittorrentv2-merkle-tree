package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	pkg "github.com/fenollp/bittorrentv2-merkle-tree"
)

func main() {
	if err := actual(); err != nil {
		log.Fatalln(err)
	}
}

func actual() (err error) {
	if len(os.Args) < 2 {
		err = errors.New("Usage: $0 <FILE>+")
		return
	}

	fmt.Println("Using block size:", pkg.BlockSize, "bytes")

	fns := os.Args[1:]
	var fds []io.Reader
	var totalBlocks int64
	for _, fn := range fns {
		fd := os.Stdin // TODO: STDIN iff fns=["-"]
		if fd, err = os.Open(fn); err != nil {
			return
		}
		defer fd.Close()

		var fi os.FileInfo
		if fi, err = fd.Stat(); err != nil {
			return
		}
		if fi.IsDir() || !fi.Mode().IsRegular() {
			continue
		}

		blocks := fi.Size() / int64(pkg.BlockSize)
		fmt.Printf("%q\n  Blocks: %d\n", fn, blocks)
		totalBlocks += blocks
		fds = append(fds, fd)
	}
	fmt.Println("Files:", len(fds))

	ctx := context.TODO()
	hashes, err := pkg.MultiBlocks(ctx, fds, totalBlocks)
	if err != nil {
		return
	}

	fmt.Println("")
	ratio := float32(100)
	ratio *= float32(len(hashes))
	ratio /= float32(totalBlocks)
	fmt.Println("In common:", len(hashes), "of", totalBlocks, "i.e.", ratio, "%")
	x := 10
	for h, c := range hashes {
		if x == 0 {
			fmt.Printf("\t...\n")
			return
		}
		fmt.Println(c, h)
		x--
	}

	return
}

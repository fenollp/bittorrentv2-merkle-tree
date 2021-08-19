package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const blockSize = 16384 // 16kiB = Bittorrentv2 block size

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
	fmt.Println("Using block size:", blockSize, "bytes")

	var totalBlocks int64
	var fds []*os.File
	for _, fn := range os.Args[1:] {
		fd := os.Stdin
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
		expectingChunks := fi.Size() / blockSize
		fmt.Println(fn, "\n  Blocks:", expectingChunks)
		totalBlocks += expectingChunks
		fds = append(fds, fd)
	}
	fmt.Println("Files:", len(fds))

	ctx := context.TODO()
	g, ctx := errgroup.WithContext(ctx)

	var aFDs uint32
	chHash := make(chan string)
	counts := make(map[string]uint, totalBlocks)
	streamers := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))

	tryClosing := func() {
		if done := atomic.LoadUint32(&aFDs); int(done) == len(fds) {
			log.Println("close(chHash)", done)
			close(chHash)
		}
	}

	for n, fd := range fds {
		n, fd := n, fd
		g.Go(func() error {
			defer func() {
				log.Println("streamed", n)
				atomic.AddUint32(&aFDs, 1)
				go tryClosing()
			}()
			if err := streamers.Acquire(ctx, 1); err != nil {
				return err
			}
			defer streamers.Release(1)
			r := bufio.NewReader(fd)
			buf := make([]byte, 0, blockSize)
			for {
				n, err := r.Read(buf[:cap(buf)])
				buf = buf[:n]
				if n == 0 {
					if err == nil {
						continue
					}
					if err == io.EOF {
						break
					}
					return err
				}
				if len(buf) < blockSize {
					break
				}
				if err != nil && err != io.EOF {
					return err
				}
				chHash <- fmt.Sprintf("%x", sha256.Sum256(buf))
			}
			return nil
		})
	}

	g.Go(func() error {
		for h := range chHash {
			counts[h]++
		}
		return nil
	})

	if err = g.Wait(); err != nil {
		return err
	}

	fmt.Println("")
	for h, c := range counts {
		if c == 1 {
			delete(counts, h)
		}
	}
	ratio := float32(100)
	ratio *= float32(len(counts))
	ratio /= float32(totalBlocks)
	fmt.Println("In common:", len(counts), "of", totalBlocks, "i.e.", ratio, "%")
	for h, c := range counts {
		fmt.Println(c, h)
	}

	return nil
}

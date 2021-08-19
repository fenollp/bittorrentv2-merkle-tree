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
	"time"

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
		expectingChunks := fi.Size() / blockSize
		fmt.Println(fn, "\n  Blocks:", expectingChunks)
		totalBlocks += expectingChunks
		fds = append(fds, fd)
	}
	fmt.Println("Files:", len(fds))

	chBytes := make(chan []byte)
	chHash := make(chan string)

	ctx := context.TODO()
	g, ctx := errgroup.WithContext(ctx)

	counts := make(map[string]uint, totalBlocks)

	maxProcs := runtime.GOMAXPROCS(0)
	if maxProcs > 1 {
		maxProcs-- // Reading 1 file blocks 1 OS thread, so leave some room.
	}
	streamers := semaphore.NewWeighted(int64(maxProcs))

	var aFDs uint32
	g.Go(func() error {
		for {
			select {
			case <-time.After(50 * time.Millisecond):
				if done := atomic.LoadUint32(&aFDs); int(done) == len(fds) {
					log.Println("close(chBytes)")
					close(chBytes)
					return nil
				}
			}
		}
	})

	for n, fd := range fds {
		n := n
		g.Go(func() error {
			defer func() {
				log.Println("streamed", n)
				streamers.Release(1)
				atomic.AddUint32(&aFDs, 1)
			}()
			if err := streamers.Acquire(ctx, 1); err != nil {
				return err
			}
			r := bufio.NewReader(fd)
			for {
				buf := make([]byte, 0, blockSize)
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
				chBytes <- buf
			}
			return nil
		})
	}

	g.Go(func() error {
		for h := range chHash {
			counts[h]++
			// log.Println(h, len(counts))
		}
		return nil
	})

	maxHashers := maxProcs / 3
	var aHashers uint32
	g.Go(func() error {
		for {
			select {
			case <-time.After(50 * time.Millisecond):
				if done := atomic.LoadUint32(&aHashers); int(done) == maxHashers {
					log.Println("close(chHash)")
					close(chHash)
					return nil
				}
			}
		}
	})

	for n := range make([]struct{}, maxHashers) {
		n := n
		g.Go(func() error {
			defer func() {
				log.Println("ending hasher", n)
				atomic.AddUint32(&aHashers, 1)
			}()
			for b := range chBytes {
				h := sha256.New()
				h.Write(b)
				chHash <- fmt.Sprintf("%x", h.Sum(nil))
			}
			return nil
		})
	}

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

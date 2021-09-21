package bittorrentv2merkletree

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"runtime"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// BlockSize sets the size of consecutive blocks to extract from files
var BlockSize = 16384 // 16kiB = Bittorrentv2 block size

// MultiBlocks identifies BlockSize-sized blocks in files with same SHA256.
func MultiBlocks(
	ctx context.Context,
	fds []io.Reader,
	maxBlocks int64,
) (
	counts map[string]uint32,
	err error,
) {
	g, ctx := errgroup.WithContext(ctx)

	var aFDs uint32
	chHash := make(chan string)
	counts = make(map[string]uint32, maxBlocks) // TODO: autogrow for STDIN
	streamers := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))

	for n, fd := range fds {
		n, fd := n, fd
		g.Go(func() error {
			defer func() {
				if done := atomic.AddUint32(&aFDs, 1); int(done) == len(fds) {
					log.Println("close(chHash)")
					close(chHash)
				} else {
					log.Printf("streamed %d/%d:\t[%d]", done, len(fds), n)
				}
			}()

			if err := streamers.Acquire(ctx, 1); err != nil {
				return err
			}
			defer streamers.Release(1)

			r := bufio.NewReader(fd)
			buf := make([]byte, 0, BlockSize)
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
				if len(buf) < BlockSize {
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
		return
	}

	for h, c := range counts {
		if c == 1 {
			delete(counts, h)
		}
	}

	return
}

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

	"golang.org/x/sync/errgroup"
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

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	chBytes := make(chan []byte)
	chHash := make(chan string)
	counts := make(map[string]uint)

	fd := os.Stdin
	fn := os.Args[1]
	if fd, err = os.Open(fn); err != nil {
		return err
	}

	g.Go(func() error {
		defer fd.Close()
		defer close(chBytes)

		buf := make([]byte, 0, blockSize)
		r := bufio.NewReader(fd)
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
				return nil
			}
			if err != nil && err != io.EOF {
				return err
			}
			chBytes <- buf
		}
		return nil
	})

	for range make([]struct{}, runtime.NumCPU()) {
		g.Go(func() error {
			for b := range chBytes {
				h := sha256.New()
				h.Write(b)
				chHash <- fmt.Sprintf("%x", h.Sum(nil))
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

	for h, c := range counts {
		if c == 1 {
			continue
		}
		fmt.Println(c, h)
	}

	return nil
}

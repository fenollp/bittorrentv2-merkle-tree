package main

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	// "golang.org/x/sync/errgroup"
)

func main() {
	if err := actual(); err != nil {
		log.Fatalln(err)
	}
}

func actual() error {
	const blockSize = 16384 // 16kiB = Bittorrentv2 block size

	type bckt = uint8
	hashes := make(map[string]bckt)

	// main <- wg 1 <- wg n

	// ctx:=context.Background()
	// g,ctx:=errgroup.WithContext(ctx)
	// hashes:=make(chan []byte)

	// g.Go(func() (err error){
	// })

	if len(os.Args) != 3 {
		return errors.New("go run split.go -- FILE")
	}

	fd := os.Stdin
	if fn := os.Args[2]; fn != "-" {
		var err error
		if fd, err = os.Open(fn); err != nil {
			return err
		}
		defer fd.Close()
	}

	r := bufio.NewReader(fd)
	for k := int64(0); true; k += blockSize {
		if err := func() error {
			log.Printf("k = %d", k)
			data := make([]byte, blockSize)

			// n, err := r.ReadAt(data[:], k)
			n, err := io.ReadFull(r, data)
			switch err {
			case nil:
			case io.EOF, io.ErrUnexpectedEOF:
				k = -1
				return nil
			default:
				return fmt.Errorf("couldn't read from byte %d (read %d): %v", k, n, err)
			}

			if n != blockSize {
				if n != 0 {
					log.Printf("skipping the last %d bytes", n)
				}
				return nil
			}

			var hashed string
			{
				h := sha256.New()
				h.Write(data[:blockSize])
				hashed = fmt.Sprintf("%x", h.Sum(nil))
			}

			if _, ok := hashes[hashed]; !ok {
				hashes[hashed] = 1
				return nil
			}
			hashes[hashed]++
			return nil
		}(); err != nil {
			return err
		}
		if k < 0 {
			break
		}
	}

	showLargerN := func(n bckt) {
		c := 0
		for _, v := range hashes {
			if v >= n {
				c++
			}
		}
		log.Printf("%d\t%d\n", n, c)
	}
	baseN := bckt(1)
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)
	baseN *= 2
	showLargerN(baseN)

	return nil
}

// func split(buf []byte, lim int) [][]byte {
// 	var chunk []byte
// 	chunks := make([][]byte, 0, len(buf)/lim+1)
// 	for len(buf) >= lim {
// 		chunk, buf = buf[:lim], buf[lim:]
// 		chunks = append(chunks, chunk)
// 	}
// 	if len(buf) > 0 {
// 		chunks = append(chunks, buf[:])
// 	}
// 	return chunks
// }

// const PackSizeLimit = 5 * 1024 * 1024

// // [a b c] -> [[a, b], [c]]
// // where (len(a)+len(b) < MAX) and (len(c) < MAX),
// // where (len(a) < MAX) and (len(b) < MAX) and (len(c) < MAX),
// // but (len(a) + len(b) + len(c)) > MAX
// func splitForJoin(chunks [][]byte, lim int) [][][]byte {
// 	var result [][][]byte
// 	for len(chunks) > 0 {
// 		for i, v := range chunks {
// 			if size+len(v) > PackSizeLimit {
// 				chunks = chunks[i:]
// 				break
// 			}
// 		}
// 		// TODO
// 	}
// 	return result
// }

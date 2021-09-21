package bittorrentv2merkletree

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMultiBlocks(t *testing.T) {
	ctx := context.Background()
	const k = 5
	blocks := k * int64(BlockSize)

	zeros := make([]byte /*blocks,*/, blocks)
	hashes, err := MultiBlocks(ctx, []io.Reader{bytes.NewReader(zeros)}, blocks)
	require.NoError(t, err)
	require.Equal(t, hashes, map[string]uint32{
		"4fe7b59af6de3b665b67788cc2f99892ab827efae3a467342b3bb4e3bc8e5bfe": k,
	})

	ones := make([]byte, 0, blocks)
	for range make([]struct{} /*,*/, blocks) {
		ones = append(ones, 1)
	}
	hashes, err = MultiBlocks(ctx, []io.Reader{bytes.NewReader(ones)}, blocks)
	require.NoError(t, err)
	require.Equal(t, hashes, map[string]uint32{
		"111ce3c2a38d83a2e4706bde4abddd509d7f8248116c6832b06745bdc349e09f": k,
	})

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rands := make([]byte, 0, blocks)
	for range make([]struct{} /*,*/, blocks) {
		b := byte(r.Intn(255))
		rands = append(rands, b)
	}
	hashes, err = MultiBlocks(ctx, []io.Reader{bytes.NewReader(rands)}, blocks)
	require.NoError(t, err)
	require.Empty(t, hashes)

	hashes, err = MultiBlocks(ctx, []io.Reader{
		bytes.NewReader(zeros),
		bytes.NewReader(ones),
		bytes.NewReader(rands),
	}, blocks)
	require.NoError(t, err)
	require.Equal(t, hashes, map[string]uint32{
		"4fe7b59af6de3b665b67788cc2f99892ab827efae3a467342b3bb4e3bc8e5bfe": k,
		"111ce3c2a38d83a2e4706bde4abddd509d7f8248116c6832b06745bdc349e09f": k,
	})
}

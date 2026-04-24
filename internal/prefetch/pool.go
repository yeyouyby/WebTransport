package prefetch

import (
	"context"
	"io"
	"sync"

	"matrix-gateway/internal/storage"
)

type BufferPool struct {
	pool sync.Pool
	size int
}

func NewBufferPool(size int) *BufferPool {
	if size <= 0 {
		size = 256 * 1024
	}
	return &BufferPool{
		size: size,
		pool: sync.Pool{New: func() any {
			b := make([]byte, size)
			return &b
		}},
	}
}

func (p *BufferPool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

func (p *BufferPool) Put(buf *[]byte) {
	if buf == nil {
		return
	}
	b := *buf
	if cap(b) < p.size {
		return
	}
	b = b[:p.size]
	*buf = b
	p.pool.Put(buf)
}

type FetchTask struct {
	Offset uint64
	Length uint32
}

type Fetcher interface {
	FetchRange(ctx context.Context, tp storage.TokenProvider, offset uint64, length uint32) (body io.ReadCloser, contentLength int64, sa storage.SAEntry, err error)
}

func BuildVideoDeepBufferTasks(startOffset uint64, chunkSize uint32, chunks int) []FetchTask {
	if chunks <= 0 {
		chunks = 20
	}
	out := make([]FetchTask, 0, chunks)
	for i := 0; i < chunks; i++ {
		offset := startOffset + uint64(i)*uint64(chunkSize)
		out = append(out, FetchTask{Offset: offset, Length: chunkSize})
	}
	return out
}

func BuildComicBurstTasks(imageOffset uint64, imageLength uint32, split int) []FetchTask {
	if split <= 0 {
		split = 10
	}
	chunk := imageLength / uint32(split)
	if chunk == 0 {
		chunk = imageLength
		split = 1
	}

	out := make([]FetchTask, 0, split)
	for i := 0; i < split; i++ {
		offset := imageOffset + uint64(i)*uint64(chunk)
		length := chunk
		if i == split-1 {
			used := uint32(i) * chunk
			length = imageLength - used
		}
		out = append(out, FetchTask{Offset: offset, Length: length})
	}
	return out
}

package commandutil

import (
	"bytes"
	"io"
)

type cappedBuffer struct {
	bytes.Buffer
	limit     int64
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if int64(b.Len()) >= b.limit {
		b.truncated = len(p) > 0
		return len(p), nil
	}
	remaining := b.limit - int64(b.Len())
	if int64(len(p)) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func (b *cappedBuffer) ReadFrom(reader io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)
	var total int64
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			_, _ = b.Write(buffer[:count])
			total += int64(count)
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

func (b *cappedBuffer) Bytes() []byte { return append([]byte(nil), b.Buffer.Bytes()...) }

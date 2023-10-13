package httpwriter

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"
)

const defaultBufferCap = 1024
const defaultBatchSize = 5
const MimeTypeApplicationJson = "application/json"
const HeaderContentType = "Content-Type"

type HttpWriterErrorFunc func(string, error)

type HttpWriterOptions struct {
	BufferCapacity int
	BatchSize      int
	ErrorFunc      HttpWriterErrorFunc
}

type HttpWriter struct {
	httpEndpoint string
	options      HttpWriterOptions
	ctx          context.Context
	ch           chan []byte
	client       *http.Client
}

func noopError(string, error) {
	// Nothing to do
}

func New(ctx context.Context, httpEndpoint string, options *HttpWriterOptions) *HttpWriter {
	opt := HttpWriterOptions{
		BufferCapacity: defaultBufferCap,
		ErrorFunc:      noopError,
		BatchSize:      defaultBatchSize,
	}
	if options != nil {
		if options.ErrorFunc != nil {
			opt.ErrorFunc = options.ErrorFunc
		}
		if options.BufferCapacity > 0 {
			opt.BufferCapacity = options.BufferCapacity
		}
		if options.BatchSize > 0 {
			opt.BatchSize = options.BatchSize
		}
	}
	s := HttpWriter{
		ctx:          ctx,
		httpEndpoint: httpEndpoint,
		options:      opt,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: false,
				MaxIdleConns:      5,
				IdleConnTimeout:   60 * time.Second,
				WriteBufferSize:   1 * 1024 * 1024, // 1MB buffer
			},
		},
		ch: make(chan []byte, opt.BufferCapacity),
	}
	go s.run()
	return &s
}

func (s *HttpWriter) Write(p []byte) (n int, err error) {
	s.ch <- bytes.Clone(p)
	return len(p), nil
}

func (s *HttpWriter) run() {
	for {
		select {
		case <-s.ctx.Done():
			// Nothing to do. Shutdown.
			return

		case msg := <-s.ch: // Wait for the a message
			const bufferSize = 256 * 1024
			byteBuf := bytes.NewBuffer(make([]byte, 0, bufferSize))
			_, _ = byteBuf.Write(msg)
			_, _ = byteBuf.Write([]byte("\n"))
			// Drain all outstanding messages from the channel in a non-blocking way.
			done := false
			count := 0
			for !done {
				select {
				case <-s.ctx.Done():
					return
				case buf, ok := <-s.ch:
					if !ok {
						s.options.ErrorFunc("Error reading continuously from channel", nil)
						continue
					}
					_, _ = byteBuf.Write(buf)
					_, _ = byteBuf.Write([]byte("\n"))
					count++
					if count >= s.options.BatchSize {
						done = true
					}
				default:
					done = true
				}
			}
			go func() {
				req, _ := http.NewRequestWithContext(s.ctx, http.MethodPost, s.httpEndpoint, byteBuf)
				req.Header.Set(HeaderContentType, MimeTypeApplicationJson)
				resp, err := s.client.Do(req)

				if err != nil {
					s.options.ErrorFunc("Error sending request", err)
					return
				}
				defer resp.Body.Close()
				if (resp.StatusCode != http.StatusOK) && (resp.StatusCode != http.StatusCreated) {
					s.options.ErrorFunc("HTTP Error "+resp.Status, errors.New(resp.Status))
					return
				}
			}()
		}
	}
}

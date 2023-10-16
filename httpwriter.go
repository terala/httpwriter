package httpwriter

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	MimeTypeApplicationJson = "application/json"
	HeaderContentType       = "Content-Type"

	defaultBufferCap       = 1024
	defaultBatchSize       = 5
	defaultMaxIdleConn     = 5
	defaultIdleConnTimeout = 30 * time.Second
	defaultWriteBufferSize = 1 * 1024 * 1024
)

type HttpWriterErrorFunc func(string, error)

type HttpWriterOptions struct {
	HttpEndpoint       string
	BufferCapacity     int
	BatchSize          int
	ErrorFunc          HttpWriterErrorFunc
	MaxIdleConnections int
	IdleConnTimeout    time.Duration
	WriteBufferSize    int
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

func defaultConfig() (*HttpWriterOptions, error) {
	opt := &HttpWriterOptions{
		HttpEndpoint:       "",
		BufferCapacity:     defaultBufferCap,
		BatchSize:          defaultBatchSize,
		ErrorFunc:          noopError,
		MaxIdleConnections: defaultMaxIdleConn,
		IdleConnTimeout:    defaultIdleConnTimeout,
		WriteBufferSize:    defaultWriteBufferSize,
	}

	err := error(nil)

	// Update base options from environment.
	if val := os.Getenv("HTTP_WRITER_ENDPOINT"); val != "" {
		u, err := url.Parse(val)
		if err != nil {
			return nil, err
		}
		opt.HttpEndpoint = u.String()
	}
	if val := os.Getenv("HTTP_WRITER_BUFFER_CAPACITY"); val != "" {
		opt.BufferCapacity, err = strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
	}
	if val := os.Getenv("HTTP_WRITER_BATCH_SIZE"); val != "" {
		opt.BatchSize, err = strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
	}
	if val := os.Getenv("HTTP_WRITER_MAX_IDLE_CONNECTIONS"); val != "" {
		opt.MaxIdleConnections, err = strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
	}
	if val := os.Getenv("HTTP_WRITER_IDLE_CONN_TIMEOUT"); val != "" {
		opt.IdleConnTimeout, err = time.ParseDuration(val)
		if err != nil {
			return nil, err
		}
	}
	if val := os.Getenv("HTTP_WRITER_WRITE_BUFFER_SIZE"); val != "" {
		opt.WriteBufferSize, err = strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
	}

	return opt, err
}

func New(ctx context.Context, options *HttpWriterOptions) (*HttpWriter, error) {
	opt, err := defaultConfig()
	if options != nil {
		if options.HttpEndpoint != "" {
			u, err := url.Parse(options.HttpEndpoint)
			if err != nil {
				return nil, err
			}
			opt.HttpEndpoint = u.String()
		}
		if options.ErrorFunc != nil {
			opt.ErrorFunc = options.ErrorFunc
		}
		if options.BufferCapacity > 0 {
			opt.BufferCapacity = options.BufferCapacity
		}
		if options.BatchSize > 0 {
			opt.BatchSize = options.BatchSize
		}
		if options.MaxIdleConnections > 0 {
			opt.MaxIdleConnections = options.MaxIdleConnections
		}
		if options.IdleConnTimeout > 0 {
			opt.IdleConnTimeout = options.IdleConnTimeout
		}
		if options.WriteBufferSize > 0 {
			opt.WriteBufferSize = options.WriteBufferSize
		}
	}
	s := HttpWriter{
		ctx:          ctx,
		httpEndpoint: opt.HttpEndpoint,
		options:      *opt,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: false,
				MaxIdleConns:      opt.MaxIdleConnections,
				IdleConnTimeout:   opt.IdleConnTimeout,
				WriteBufferSize:   opt.WriteBufferSize,
			},
		},
		ch: make(chan []byte, opt.BufferCapacity),
	}
	go s.run()
	return &s, err
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

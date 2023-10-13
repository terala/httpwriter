package httpwriter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/maargenton/go-testpredicate/pkg/verify"
	"github.com/stretchr/testify/suite"
	"github.com/terala/httpwriter"
)

const (
	jsonLinesCapacity = 1024
)

type jsonLine map[string]interface{}

type HttpWriterTestSuite struct {
	suite.Suite

	ctx        context.Context
	cancel     context.CancelFunc
	mockServer *httptest.Server
	jsonLines  []jsonLine
	rwmu       sync.RWMutex
}

func (s *HttpWriterTestSuite) SetupTest() {
	s.jsonLines = make([]jsonLine, 0, jsonLinesCapacity)
	s.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// It must be POST
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get(httpwriter.HeaderContentType) != httpwriter.MimeTypeApplicationJson {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		contentType := r.Header.Get(httpwriter.HeaderContentType)
		switch contentType {
		case httpwriter.MimeTypeApplicationJson:
			dec := json.NewDecoder(r.Body)
			for {
				var line jsonLine
				err := dec.Decode(&line)
				if err == io.EOF {
					break
				} else if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				s.rwmu.Lock()
				s.jsonLines = append(s.jsonLines, line)
				s.rwmu.Unlock()
			}
		// case httpwriter.MimeTypeApplicationOctetStream:
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
	}))
	s.ctx, s.cancel = context.WithCancel(context.Background())

}

func (s *HttpWriterTestSuite) TearDownTest() {
	s.jsonLines = nil
	s.mockServer.Close()
}

func (s *HttpWriterTestSuite) Test_write_one_line() {
	// Arrange
	const key1 = "key1"
	const value1 = "value1"
	const key2 = "key2"
	const value2 = "value2"

	line := fmt.Sprintf("{\"%s\":\"%s\",\"%s\":\"%s\"}", key1, value1, key2, value2)
	w := httpwriter.New(s.ctx, s.mockServer.URL, nil)

	// Act
	_, _ = w.Write([]byte(line))
	time.Sleep(25 * time.Millisecond)
	s.cancel()

	// Assert
	verify.That(s.T(), len(s.jsonLines)).IsEqualTo(1)
	verify.That(s.T(), s.jsonLines[0]).MapKeys().IsEqualSet([]string{key1, key2})
	verify.That(s.T(), s.jsonLines[0]).MapValues().IsEqualSet([]string{value1, value2})
}

func (s *HttpWriterTestSuite) Test_write_multiple_lines() {
	// Arrange

	var batchTests = []struct {
		name      string
		batchSize int
	}{
		{"BatchSize: 5", 5},
		{"BatchSize: 25", 25},
		{"BatchSize: 50", 50},
		{"BatchSize: 100", 100},
		{"BatchSize: 500", 500},
		{"BatchSize: 100", 1000},
	}
	for _, bat := range batchTests {
		s.T().Run(bat.name, func(t *testing.T) {

			var bufferTests = []struct {
				name       string
				bufferSize int
			}{
				{"BufferCapacity: 0", 0},
				{"BufferCapacity: 25", 25},
				{"BufferCapacity: 50", 50},
				{"BufferCapacity: 100", 100},
				{"BufferCapacity: 250", 250},
				{"BufferCapacity: 500", 500},
				{"BufferCapacity: 1000", 1000},
			}
			for _, bt := range bufferTests {
				s.T().Run(bt.name, func(t *testing.T) {
					var tests = []struct {
						name  string
						count int
					}{
						{name: "100 lines", count: 100},
						{name: "500 lines", count: 500},
						{name: "1000 lines", count: 1000},
						{name: "5000 lines", count: 5000},
						{name: "10,000 lines", count: 10000},
					}
					for _, tt := range tests {
						s.T().Run(fmt.Sprintf("%s/%s/%s", bat.name, bt.name, tt.name), func(t *testing.T) {
							s.runTest(tt.count, bt.bufferSize, bat.batchSize)
						})
					}

				})
			}
		})
	}
}

func (s *HttpWriterTestSuite) runTest(count int, bufferSize int, batchSize int) {
	s.SetupTest()
	defer s.TearDownTest()

	// Arrange
	const keyName = "counter"
	options := httpwriter.HttpWriterOptions{
		BufferCapacity: bufferSize,
		BatchSize:      batchSize,
	}
	w := httpwriter.New(s.ctx, s.mockServer.URL, &options)

	// Act
	for i := 0; i < count; i++ {
		line := fmt.Sprintf(`{"%s":"%d"}`, keyName, i)
		_, _ = w.Write([]byte(line))
	}
	// Wait max 2 seconds for all lines to be written
	s.maxSleep(count, 2*time.Second)
	s.cancel()

	// Assert
	s.Assert().Equal(len(s.jsonLines), count)
	verify.That(s.T(), len(s.jsonLines)).IsEqualTo(count)
	// sort.Slice(s.jsonLines, func(i, j int) bool {
	// 	left := s.jsonLines[i][keyName]
	// 	right := s.jsonLines[j][keyName]
	// 	leftVal, _ := strconv.Atoi(left.(string))
	// 	rightVal, _ := strconv.Atoi(right.(string))
	// 	return leftVal < rightVal
	// })
	// for i := 0; i < count; i++ {
	// 	verify.That(s.T(), s.jsonLines[i][keyName]).IsEqualTo(fmt.Sprintf("%d", i))
	// }
}

func (s *HttpWriterTestSuite) maxSleep(count int, maxTime time.Duration) {
	const sleepTime = 5 * time.Millisecond
	slept := time.Duration(0)
	done := false
	for !done {
		s.rwmu.RLock()
		l := len(s.jsonLines)
		s.rwmu.RUnlock()
		if l < count {
			time.Sleep(sleepTime)
			slept += sleepTime
		}
		if (l >= count) || (slept >= maxTime) {
			done = true
		}
	}
}

func (s *HttpWriterTestSuite) Test_error_func_is_invoked_upon_errors() {
	var tests = []struct {
		name string
		url  string
	}{
		{name: "invalid url", url: "https://invalid.url/that/will/never/work"},
		{name: "unparsable url", url: "invalid url/ that/ will not parse"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.SetupTest()
			defer s.TearDownTest()
			// Arrange
			err := make(chan error)
			msg := ""
			options := httpwriter.HttpWriterOptions{
				ErrorFunc: func(s string, e error) {
					msg = s
					err <- e
				},
			}

			// Act
			w := httpwriter.New(s.ctx, tt.url, &options)
			_, _ = w.Write([]byte("{\"key\":\"value\"}"))
			er := <-err // Wait until error is available.
			s.cancel()

			// Assert
			verify.That(s.T(), er).IsNotNil()
			verify.That(s.T(), msg).IsNotEmpty()
		})
	}
}

func (s *HttpWriterTestSuite) Test_slog_integration() {
	// Arrange
	w := httpwriter.New(s.ctx, s.mockServer.URL, nil)
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(jsonHandler)
	msg := "SLog Integration"
	count := 5

	// Act
	for i := 0; i < count; i++ {
		logger.Info(msg, slog.Int("counter", i))
	}
	s.maxSleep(count, 1*time.Second)
	s.cancel()

	// Assert
	verify.That(s.T(), len(s.jsonLines)).IsEqualTo(count)
	for i := 0; i < count; i++ {
		verify.That(s.T(), s.jsonLines[i]["msg"]).IsEqualTo(msg)
		verify.That(s.T(), s.jsonLines[i]["level"]).IsEqualTo("INFO")
		verify.That(s.T(), s.jsonLines[i]["counter"]).IsEqualTo(i)
	}
}

func (s *HttpWriterTestSuite) Test_is_cancellable_by_context() {
	// Arrange
	options := httpwriter.HttpWriterOptions{
		BatchSize:      5,
		BufferCapacity: 250,
	}
	w := httpwriter.New(s.ctx, s.mockServer.URL, &options)
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(jsonHandler)
	msg := "Large number of lines"
	count := 1000

	// Act
	for i := 0; i < count; i++ {
		logger.Info(msg, slog.Int("counter", i))
	}
	s.cancel()

	// Assert
	s.T().Log("jsonLines count:", len(s.jsonLines))
	verify.That(s.T(), len(s.jsonLines)).IsLessOrEqualTo(count)
}

func (s *HttpWriterTestSuite) Test_external_slog_roundtrip() {
	// Arrange
	const count = 25
	err := make(chan error)
	errorMsg := ""
	options := httpwriter.HttpWriterOptions{
		ErrorFunc: func(s string, e error) {
			errorMsg = s
			err <- e
		},
		BufferCapacity: 3,
		BatchSize:      5,
	}
	w := httpwriter.New(s.ctx, "http://localhost:8888/", &options)
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(jsonHandler)
	msg := "SLog -> FluentBit"

	// Act
	for i := 0; i < count; i++ {
		logger.Info(fmt.Sprintf("%s : %d", msg, i), slog.Int("counter", i))
	}
	time.Sleep(1 * time.Second)
	s.cancel()

	// Assert
	verify.That(s.T(), errorMsg).IsEmpty()
}

func (s *HttpWriterTestSuite) Test_errors_are_reported_for_non_201_response() {
	// Arrange
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.WriteHeader(http.StatusBadRequest)
	}))

	msg := ""
	err := errors.New("no error")
	options := httpwriter.HttpWriterOptions{
		ErrorFunc: func(s string, e error) {
			msg = s
			err = e
		},
	}
	w := httpwriter.New(s.ctx, mockServer.URL, &options)
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(jsonHandler)

	// Act
	logger.Info("SLog -> FluentBit")
	time.Sleep(25 * time.Millisecond)
	s.cancel()

	// Assert
	verify.That(s.T(), msg).IsNotEmpty()
	verify.That(s.T(), err).IsNotNil()
}

func Test_HttpWriter(t *testing.T) {
	suite.Run(t, new(HttpWriterTestSuite))
}

package httpwriter

import (
	"strconv"
	"testing"
	"time"

	"github.com/maargenton/go-testpredicate/pkg/verify"
)

func Test_http_endpoint_via_env(t *testing.T) {
	// Arrange
	u := "http://localhost:8888/"
	t.Setenv("HTTP_WRITER_ENDPOINT", u)

	// Act
	opt, err := defaultConfig()

	// Assert
	verify.That(t, err).IsNil()
	verify.That(t, opt.HttpEndpoint).IsEqualTo(u)
}

func Test_buffer_capacity_via_env(t *testing.T) {
	// Arrange
	val := 3
	t.Setenv("HTTP_WRITER_BUFFER_CAPACITY", strconv.Itoa(val))

	// Act
	opt, err := defaultConfig()

	// Assert
	verify.That(t, err).IsNil()
	verify.That(t, opt.BufferCapacity).IsEqualTo(val)
}

func Test_batch_size_via_env(t *testing.T) {
	// Arrange
	val := 100
	t.Setenv("HTTP_WRITER_BATCH_SIZE", strconv.Itoa(val))

	// Act
	opt, err := defaultConfig()

	// Assert
	verify.That(t, err).IsNil()
	verify.That(t, opt.BatchSize).IsEqualTo(val)
}

func Test_max_idle_conn_via_env(t *testing.T) {
	// Arrange
	val := 10
	t.Setenv("HTTP_WRITER_MAX_IDLE_CONNECTIONS", strconv.Itoa(val))

	// Act
	opt, err := defaultConfig()

	// Assert
	verify.That(t, err).IsNil()
	verify.That(t, opt.MaxIdleConnections).IsEqualTo(val)
}

func Test_idle_timeout_via_env(t *testing.T) {
	// Arrange
	val := 10 * time.Second
	t.Setenv("HTTP_WRITER_IDLE_CONN_TIMEOUT", val.String())

	// Act
	opt, err := defaultConfig()

	// Assert
	verify.That(t, err).IsNil()
	verify.That(t, opt.IdleConnTimeout).IsEqualTo(val)
}

func Test_write_buffer_size_via_env(t *testing.T) {
	// Arrange
	val := 250
	t.Setenv("HTTP_WRITER_WRITE_BUFFER_SIZE", strconv.Itoa(val))

	// Act
	opt, err := defaultConfig()

	// Assert
	verify.That(t, err).IsNil()
	verify.That(t, opt.WriteBufferSize).IsEqualTo(val)
}

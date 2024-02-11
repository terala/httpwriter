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

func Test_bad_env_vars(t *testing.T) {
	// Arrange
	var tests = []struct {
		name       string
		env_name   string
		env_value  string
		err_string string
	}{
		{"invalid_buffer_capcity", "HTTP_WRITER_BUFFER_CAPACITY", "invalid", "strconv.Atoi"},
		{"invalid_batch_size", "HTTP_WRITER_BATCH_SIZE", "invalid", "strconv.Atoi"},
		{"invalid_max_idle_conn", "HTTP_WRITER_MAX_IDLE_CONNECTIONS", "invalid", "strconv.Atoi"},
		{"invalid_writer_buffer_size", "HTTP_WRITER_WRITE_BUFFER_SIZE", "invalid", "strconv.Atoi"},
		{"invalid_idle_conn_timeout", "HTTP_WRITER_IDLE_CONN_TIMEOUT", "invalid", "time: invalid duration"},
	}
	for _, bat := range tests {
		t.Run(bat.name, func(tt *testing.T) {
			// Arrange
			tt.Setenv(bat.env_name, bat.env_value)

			// Act
			_, err := defaultConfig()

			// Assert
			verify.That(tt, err).IsNotNil()
			verify.That(tt, err.Error()).StartsWith(bat.err_string)
		})
	}
}

package httpwriter

import (
	"reflect"
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
		{"invalid_url", "HTTP_WRITER_ENDPOINT", "invalid url/ that/ will not parse", "parse"},
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

func Test_set_via_env(t *testing.T) {
	// Arrange
	var tests = []struct {
		name      string
		env_name  string
		fieldName string
		value     int
	}{
		{"write_buffer_size", "HTTP_WRITER_WRITE_BUFFER_SIZE", "WriteBufferSize", 250},
		{"buffer_capacity", "HTTP_WRITER_BUFFER_CAPACITY", "BufferCapacity", 3},
		{"batch_size", "HTTP_WRITER_BATCH_SIZE", "BatchSize", 100},
		{"max_idle_connections", "HTTP_WRITER_MAX_IDLE_CONNECTIONS", "MaxIdleConnections", 10},
	}

	// Act
	for _, bat := range tests {
		t.Run(bat.name, func(tt *testing.T) {
			// Arrange
			tt.Setenv(bat.env_name, strconv.Itoa(bat.value))

			// Act
			opt, err := defaultConfig()

			// Assert
			valRef := reflect.Indirect(reflect.ValueOf(opt))
			fld := valRef.FieldByName(bat.fieldName)
			val := fld.Int()
			verify.That(tt, err).IsNil()
			verify.That(tt, val).IsEqualTo(bat.value)
		})
	}
}

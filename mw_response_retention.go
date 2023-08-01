// based on echo middleware ResponseRetention https://github.com/labstack/echo/blob/master/middleware/body_dump.go
package main

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	// ResponseRetentionConfig defines the config for ResponseRetention middleware.
	ResponseRetentionConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper middleware.Skipper
	}

	responseRetainer struct {
		io.Writer
		http.ResponseWriter
	}
)

var (
	// DefaultResponseRetentionConfig is the default ResponseRetention middleware config.
	DefaultResponseRetentionConfig = ResponseRetentionConfig{
		Skipper: middleware.DefaultSkipper,
	}
)

// ResponseRetention returns a ResponseRetention middleware.
func ResponseRetention() echo.MiddlewareFunc {
	c := DefaultResponseRetentionConfig

	return ResponseRetentionWithConfig(c)
}

// ResponseRetentionWithConfig returns a ResponseRetention middleware with config.
// See: `ResponseRetention()`.
func ResponseRetentionWithConfig(config ResponseRetentionConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultResponseRetentionConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			// req := c.Request()
			// res := c.Response()

			// see https://github.com/labstack/echo/blob/master/middleware/body_dump.go#L76
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &responseRetainer{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer

			if err = next(c); err != nil {
				c.Error(err)
			}

			return nil
		}
	}
}

func (w *responseRetainer) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseRetainer) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *responseRetainer) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *responseRetainer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

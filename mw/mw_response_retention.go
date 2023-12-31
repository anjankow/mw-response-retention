// based on echo middleware BodyDump https://github.com/labstack/echo/blob/master/middleware/body_dump.go
package mw

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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

		// ResponseStorage provides Store and Retrieve from cache functionality
		ResponseStorage
	}

	// responseRetainer is used to access response bytes
	responseRetainer struct {
		io.Writer
		http.ResponseWriter
	}

	ResponseStorage interface {
		Store(ctx context.Context, key string, response RetainedResponse) error
		Retrieve(ctx context.Context, key string) (RetainedResponse, error)
	}

	RetainedResponse struct {
		Body       []byte
		Header     http.Header
		StatusCode int
	}
)

var (
	ErrNotRetained  = errors.New("response is not in cache")
	ErrWriteFailure = errors.New("failed to write the response")
)

// ResponseRetentionWithConfig returns a ResponseRetention middleware with config.
func ResponseRetentionWithConfig(config ResponseRetentionConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}

	if config.ResponseStorage == nil {
		panic("ResponseStorage implementation is missing")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request()
			res := c.Response()

			if rresp, err := tryRetrieve(c, config.ResponseStorage); err == nil {
				// we found the cached response!
				fmt.Println("using retrieved response: ", rresp.Header.Get(echo.HeaderXRequestID))
				return writeResponse(c, rresp)
			} // else call the next() handler and store the response

			// see https://github.com/labstack/echo/blob/master/middleware/body_dump.go#L76
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &responseRetainer{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer

			if err = next(c); err != nil {
				c.Error(err)
			}

			key, err := makeCacheKey(req)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			rresp := RetainedResponse{
				Body:       resBody.Bytes(),
				Header:     res.Header().Clone(),
				StatusCode: res.Status,
			}

			fmt.Println("storing the response : ", rresp.Header.Get(echo.HeaderXRequestID))
			if err := config.ResponseStorage.Store(req.Context(), key, rresp); err != nil {
				fmt.Println("Failed to store response: ", err)
			}

			return nil
		}
	}
}

func makeCacheKey(req *http.Request) (string, error) {
	// TODO: remove requestID and other unique per request values from headers

	var reqData = struct {
		URL     string      // assures that response is linked to a specific path
		Method  string      // assures that method will match
		Headers http.Header // enriched with auth token assures that only the initial requestor can access it
	}{
		URL:     req.URL.String(),
		Method:  req.Method,
		Headers: req.Header.Clone(),
	}

	// your algorithm of choice
	h := sha256.New()
	// reqData will be encoded to json and written to h
	e := json.NewEncoder(h)
	if err := e.Encode(reqData); err != nil {
		return "", fmt.Errorf("failed to encode reqData: %w", err)
	}

	// sha256 hash
	reqKey := fmt.Sprintf("%x", h.Sum(nil))
	return reqKey, nil
}

func tryRetrieve(c echo.Context, cache ResponseStorage) (RetainedResponse, error) {
	req := c.Request()
	key, err := makeCacheKey(req)
	if err != nil {
		return RetainedResponse{}, err
	}

	// Retrieve the cached response using the key
	return cache.Retrieve(req.Context(), key)
}

func writeResponse(c echo.Context, rresp RetainedResponse) error {
	res := c.Response()

	// set retrieved headers
	for key, values := range rresp.Header {
		res.Header().Del(key)
		for _, value := range values {
			res.Header().Add(key, value)
		}
	}

	if len(rresp.Body) > 0 {

		// The first call to Write will trigger an implicit
		// WriteHeader(http.StatusOK), see https://github.com/labstack/echo/blob/master/response.go#L50.
		n, err := res.Write(rresp.Body)
		if err != nil || n != len(rresp.Body) {
			return ErrWriteFailure
		}
	} else {
		// In case of no body, write the retained status code.
		res.WriteHeader(rresp.StatusCode)
	}

	return nil
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

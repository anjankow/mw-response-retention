package main

import (
	"context"
	"mw-response-retainer/mw"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getFish(c echo.Context) error {
	query := c.Request().URL.Query()
	if weird := query.Get("weird"); weird == "true" {
		// weird fish requested, there you go!
		return c.String(http.StatusOK, "barreleye, just google it")
	}

	return c.String(http.StatusOK, "herrings and sardines")
}

func getBirds(c echo.Context) error {
	return c.String(http.StatusOK, "owls (are not what they seem)")
}

func main() {
	e := echo.New()

	// routing
	e.GET("/api/animals/fish", getFish)
	e.GET("/api/animals/birds", getBirds)

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${path} -- ${time_rfc3339} -- latency: ${latency_human}` + "\n",
	}))

	// add requestID to response headers before response storage
	e.Use(middleware.RequestID())

	// ResponseStorage implementation using sync.Map underneath
	respStorage := &RespStorage{}
	e.Use(mw.ResponseRetentionWithConfig(mw.ResponseRetentionConfig{
		ResponseStorage: respStorage,
	}))

	e.Logger.Fatal(e.Start(":1323"))
}

type RespStorage struct {
	storage sync.Map
}

func (s *RespStorage) Store(ctx context.Context, key string, response mw.RetainedResponse) error {
	s.storage.Store(key, response)
	return nil
}

func (s *RespStorage) Retrieve(ctx context.Context, key string) (mw.RetainedResponse, error) {
	resp, ok := s.storage.Load(key)
	if !ok {
		return mw.RetainedResponse{}, mw.ErrNotRetained
	}

	return resp.(mw.RetainedResponse), nil
}

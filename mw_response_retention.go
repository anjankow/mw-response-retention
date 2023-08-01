// based on echo middleware implementation https://github.com/labstack/echo/blob/master/middleware
package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ResponseRetentionConfig defines the config for ResponseRetention middleware.
type ResponseRetentionConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper middleware.Skipper
}

// ResponseRetentionWithConfig returns a Timeout middleware with config.
func ResponseRetentionWithConfig(config ResponseRetentionConfig) echo.MiddlewareFunc {
	return config.ToMiddleware()
}

// ToMiddleware converts Config to middleware.
func (config ResponseRetentionConfig) ToMiddleware() echo.MiddlewareFunc {

	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			return next(c)
		}
	}
}

package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
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

	e.Logger.Fatal(e.Start(":1323"))
}

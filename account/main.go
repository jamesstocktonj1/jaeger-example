package main

import (
	"sync"

	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var bookingMap sync.Map

func main() {
	bookingMap = sync.Map{}

	s := echo.New()
	s.Use(middleware.Logger())
	s.Use(middleware.Recover())

	closer := jaegertracing.New(s, nil)
	defer closer.Close()

	g := s.Group("/account")
	g.GET("/login", loginHandler)
	g.GET("/booking/:id", bookingGetHandler)
	g.PUT("/booking/:id", bookingPutHandler)

	s.Logger.Fatal(s.Start(":8080"))
}

func loginHandler(c echo.Context) error {
	return c.JSON(200, map[string]string{"message": "login"})
}

func bookingGetHandler(c echo.Context) error {
	booking, ok := bookingMap.Load(c.Param("id"))

	if !ok || booking == nil {
		return c.JSON(404, map[string]string{"message": "booking not found"})
	}

	return c.JSON(200, map[string]any{"message": booking})
}

func bookingPutHandler(c echo.Context) error {
	booking := map[string]string{"id": c.Param("id")}

	bookingMap.Store(c.Param("id"), booking)

	return c.JSON(200, map[string]string{"message": "booking created"})
}

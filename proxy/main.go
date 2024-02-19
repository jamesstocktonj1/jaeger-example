package main

import (
	"fmt"

	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	s := echo.New()
	s.Use(middleware.Logger())
	s.Use(middleware.Recover())

	closer := jaegertracing.New(s, nil)
	defer closer.Close()

	config := middleware.ProxyConfig{
		Balancer: &ProxyForwarder{},
		RetryFilter: func(c echo.Context, err error) bool {
			sp := jaegertracing.CreateChildSpan(c, "Proxy Retry Filter")
			defer sp.Finish()

			sp.SetTag("proxy_error", err)
			return true
		},
		ErrorHandler: func(c echo.Context, err error) error {
			sp := jaegertracing.CreateChildSpan(c, "Proxy Error Handler")
			defer sp.Finish()

			sp.SetTag("proxy_error", err)

			sp.SetTag("error", true)
			sp.LogKV("error.message", err.Error())
			return fmt.Errorf("retry count exceeded: %s", err.Error())
		},
	}
	s.Use(middleware.ProxyWithConfig(config))

	s.Logger.Fatal(s.Start(":8080"))
}

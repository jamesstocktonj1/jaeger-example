package main

import (
	"net/url"
	"regexp"

	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ProxyForwarder struct {
	middleware.ProxyBalancer
}

func (p *ProxyForwarder) AddTarget(target *middleware.ProxyTarget) bool {
	return true
}

func (p *ProxyForwarder) RemoveTarget(name string) bool {
	return true
}

func (p *ProxyForwarder) Next(c echo.Context) *middleware.ProxyTarget {
	sp := jaegertracing.CreateChildSpan(c, "Proxy Forwarder")
	defer sp.Finish()

	account, _ := url.Parse("http://account:8080")
	booking, _ := url.Parse("http://booking:8080")

	isAccount, _ := regexp.MatchString("/account/.*", c.Request().URL.Path)
	isBooking, _ := regexp.MatchString("/booking/.*", c.Request().URL.Path)
	if isAccount {
		sp.SetTag("proxy_target", account)
		return &middleware.ProxyTarget{
			URL: account,
		}
	} else if isBooking {
		sp.SetTag("proxy_target", account)
		return &middleware.ProxyTarget{
			URL: booking,
		}
	} else {
		sp.SetTag("proxy_error", "no target found for path "+c.Request().URL.Path)
		sp.SetTag("error", true)
		return nil
	}
}

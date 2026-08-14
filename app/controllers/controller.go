package controllers

import (
	"net/http"
	"time"

	// "github.com/labstack/echo"
	"github.com/labstack/echo/v4"

	"github.com/Wanjie-Ryan/token-bucket/app/connections"
	"github.com/Wanjie-Ryan/token-bucket/app/ratelimiter"
)

type Controller struct {
	FixedWindowLimiter *ratelimiter.FixedWindowLimiter
	NaiveRedisLimiter  ratelimiter.Limiter
}

func NewController() *Controller {
	return &Controller{
		FixedWindowLimiter: ratelimiter.NewFixedWindowLimiter(5, time.Second),
		NaiveRedisLimiter:  ratelimiter.NewNaiveRedisLimiterClient(connections.RedisClient(), 5, time.Second),
	}
}

type checkRequest struct {
	ClientKey string `json:"client_key"`
}

type checkResponse struct {
	Allowed   bool `json:"allowed"`
	Remaining int  `json:"remaining"`
}

func (ctl *Controller) CheckFixedWindow(c echo.Context) error {
	var req checkRequest

	if err := c.Bind(&req); err != nil || req.ClientKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "client_key is required"})
	}

	allowed, remaining := ctl.FixedWindowLimiter.Allow(req.ClientKey)

	return c.JSON(http.StatusOK, checkResponse{
		Allowed:   allowed,
		Remaining: remaining,
	})
}

func (ctl *Controller) CheckNaiveRedis(c echo.Context) error {
	var req checkRequest

	if err := c.Bind(&req); err != nil || req.ClientKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "client_key is required"})

	}

	allowed, remaining := ctl.NaiveRedisLimiter.Allow(req.ClientKey)

	return c.JSON(http.StatusOK, checkResponse{
		Allowed:   allowed,
		Remaining: remaining,
	})
}

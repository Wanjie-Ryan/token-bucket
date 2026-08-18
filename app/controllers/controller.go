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
	TokenBucketLimiter ratelimiter.Limiter
}

func NewController() *Controller {
	return &Controller{
		FixedWindowLimiter: ratelimiter.NewFixedWindowLimiter(5, time.Second),
		NaiveRedisLimiter:  ratelimiter.NewNaiveRedisLimiterClient(connections.RedisClient(), 5, time.Second),
		TokenBucketLimiter: ratelimiter.NewTokenBucketLimiter(connections.RedisClient(), 5, 5.0),
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

	allowed, remaining := ctl.FixedWindowLimiter.Allow(c.Request().Context(), req.ClientKey)

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

	allowed, remaining := ctl.NaiveRedisLimiter.Allow(c.Request().Context(), req.ClientKey)

	return c.JSON(http.StatusOK, checkResponse{
		Allowed:   allowed,
		Remaining: remaining,
	})
}

func (ctl *Controller) CheckTokenBucket(c echo.Context) error {
	var req checkRequest
	if err := c.Bind(&req); err != nil || req.ClientKey == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "client_key is required"})
	}

	allowed, remaining := ctl.TokenBucketLimiter.Allow(c.Request().Context(), req.ClientKey)

	return c.JSON(http.StatusOK, checkResponse{
		Allowed:   allowed,
		Remaining: remaining,
	})
}

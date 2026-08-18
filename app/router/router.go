package router

import (
	"log"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Wanjie-Ryan/token-bucket/app/connections"
	"github.com/Wanjie-Ryan/token-bucket/app/controllers"
	"github.com/Wanjie-Ryan/token-bucket/app/tracing"
)

type App struct {
	E          *echo.Echo
	Controller *controllers.Controller
}

func (a *App) Initialize() {
	tracing.Init()
	connections.InitRedis()
	a.E = echo.New()
	a.E.Use(middleware.Recover())
	a.E.Use(echoprometheus.NewMiddleware("token_bucket"))
	a.E.Use(middleware.RequestLogger())
	a.Controller = controllers.NewController()
	a.registerRoutes()
}

func (a *App) registerRoutes() {
	a.E.POST("/check/fixed-window", a.Controller.CheckFixedWindow)
	a.E.POST("/check/naive-redis", a.Controller.CheckNaiveRedis)
	a.E.POST("/check/token-bucket", a.Controller.CheckTokenBucket)
	a.E.GET("/metrics", echoprometheus.NewHandler())
}

func (a *App) Run() {
	log.Fatal(a.E.Start(":8081"))
}

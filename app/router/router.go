package router

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Wanjie-Ryan/token-bucket/app/connections"
	"github.com/Wanjie-Ryan/token-bucket/app/controllers"
)

type App struct {
	E          *echo.Echo
	Controller *controllers.Controller
}

func (a *App) Initialize() {
	connections.InitRedis()
	a.E = echo.New()
	a.E.Use(middleware.Recover())
	a.Controller = controllers.NewController()
	a.registerRoutes()
}

func (a *App) registerRoutes() {
	a.E.POST("/check/fixed-window", a.Controller.CheckFixedWindow)
	a.E.POST("/check/naive-redis", a.Controller.CheckNaiveRedis)
	a.E.POST("/check/token-bucket", a.Controller.CheckTokenBucket)
}

func (a *App) Run() {
	log.Fatal(a.E.Start(":8081"))
}

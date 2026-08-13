package router

import (
	"log"

	"github.com/labstack/echo/v4"

	"github.com/Wanjie-Ryan/token-bucket/app/controllers"
)

type App struct {
	E          *echo.Echo
	Controller *controllers.Controller
}

func (a *App) Initialize() {
	a.E = echo.New()
	a.Controller = controllers.NewController()
	a.registerRoutes()
}

func (a *App) registerRoutes() {
	a.E.POST("/check/fixed-window", a.Controller.CheckFixedWindow)
}

func (a *App) Run() {
	log.Fatal(a.E.Start(":8081"))
}

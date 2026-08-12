package main

import "github.com/Wanjie-Ryan/token-bucket/app/router"

func main() {
	var a router.App
	a.Initialize()
	a.Run()
}

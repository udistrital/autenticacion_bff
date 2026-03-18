package main

import (
	beego "github.com/beego/beego/v2/server/web"

	_ "github.com/udistrital/autenticacion_bff/routers"
)

func main() {
	beego.Run()
}

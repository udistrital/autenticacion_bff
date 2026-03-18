package main

import (
	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/udistrital/autenticacion_bff/routers"
	"github.com/udistrital/autenticacion_bff/utils_oas/apistatus"
	"github.com/udistrital/autenticacion_bff/utils_oas/auditoria"
	"github.com/udistrital/autenticacion_bff/utils_oas/customerror"
)

func main() {
	beego.ErrorController(&customerror.CustomErrorController{})
	apistatus.Init()
	auditoria.InitMiddleware()
	beego.Run()
}

package main

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
	_ "github.com/udistrital/autenticacion_bff/routers"
	"github.com/udistrital/autenticacion_bff/utils_oas/apistatus"
	"github.com/udistrital/autenticacion_bff/utils_oas/auditoria"
	"github.com/udistrital/autenticacion_bff/utils_oas/customerror"
)

func main() {
	allowedOrigins := []string{
		"http://localhost:4200",
		"http://127.0.0.1:4200",
		"https://*.udistrital.edu.co",
	}

	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
	}))

	beego.ErrorController(&customerror.CustomErrorController{})
	apistatus.Init()
	auditoria.InitMiddleware()
	beego.Run()
}

package routers

import (
	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/autenticacion_bff/controllers"
)

func init() {
	beego.Router("/v1/auth/login", &controllers.AuthController{}, "get:Login")
	beego.Router("/v1/auth/login-url", &controllers.AuthController{}, "get:LoginURL")
	beego.Router("/api/v1/auth/callback", &controllers.AuthController{}, "get:Callback")
	beego.Router("/v1/auth/logout", &controllers.AuthController{}, "get:Logout")
	beego.Router("/v1/me", &controllers.MeController{}, "get:GetMe")
}

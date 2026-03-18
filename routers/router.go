package routers

import (
	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/autenticacion_bff/controllers"
)

func init() {
	beego.Router("/auth", &controllers.AuthController{})
	beego.Router("/me", &controllers.MeController{})
}

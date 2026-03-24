package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/services"
)

type MeController struct {
	beego.Controller
}

func (c *MeController) GetMe() {
	cookieName := beego.AppConfig.DefaultString("session_cookie_name", "id_sesion")
	cookie, err := c.Ctx.Request.Cookie(cookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		c.CustomAbort(401, "Sesión no encontrada")
		return
	}

	keycloak := services.NewKeycloakService()
	sesion, err := services.GetSesionActual(cookie.Value, keycloak)
	if err != nil {
		c.CustomAbort(401, err.Error())
		return
	}

	c.Data["json"] = map[string]interface{}{
		"id_usuario":         sesion.IDUsuario,
		"correo_electronico": sesion.CorreoElectronico,
		"nombre_usuario":     sesion.NombreUsuario,
	}
	_ = c.ServeJSON()
}

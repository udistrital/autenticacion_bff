package controllers

import (
	"errors"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/services"
	"github.com/udistrital/autenticacion_bff/utils_oas/errorctrl"
)

// MeController gestiona la consulta de la sesión autenticada del BFF.
type MeController struct {
	beego.Controller
}

// GetMe ...
// @Title GetMe
// @Description Retorna la información básica de la sesión autenticada actual
// @Success 200 {object} map[string]interface{}
// @Failure 401 sesión no encontrada, inválida o expirada
// @router /me [get]
func (c *MeController) GetMe() {
	defer errorctrl.ErrorControlController(c.Controller, "MeController/GetMe")

	cookieName := beego.AppConfig.DefaultString("session_cookie_name", "id_sesion")
	cookie, err := c.Ctx.Request.Cookie(cookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		if err == nil {
			err = errors.New("sesión no encontrada")
		}
		panic(errorctrl.Error(`GetMe - c.Ctx.Request.Cookie(cookieName)`, err, "401"))
	}

	sesion, err := services.GetSesionActual(cookie.Value)
	if err != nil {
		panic(errorctrl.Error("GetMe - services.GetSesionActual(cookie.Value)", err, "401"))
	}
	if sesion == nil {
		panic(errorctrl.Error("GetMe - services.GetSesionActual(cookie.Value)", errors.New("sesión inválida o expirada"), "401"))
	}

	c.Data["json"] = map[string]interface{}{
		"cliente_id":         sesion.ClienteID,
		"id_usuario":         sesion.IDUsuario,
		"correo_electronico": sesion.CorreoElectronico,
		"nombre_usuario":     sesion.NombreUsuario,
	}
	_ = c.ServeJSON()
}

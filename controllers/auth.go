package controllers

import (
	"net/http"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/services"
)

// AuthController operations for Auth
type AuthController struct {
	beego.Controller
}

func (c *AuthController) Login() {
	authURL := services.NewKeycloakService().BuildLoginURL("estado-inicial")
	c.Redirect(authURL, http.StatusTemporaryRedirect)
}

func (c *AuthController) LoginURL() {
	authURL := services.NewKeycloakService().BuildLoginURL("estado-inicial")
	c.Data["json"] = map[string]string{
		"auth_url": authURL,
	}
	_ = c.ServeJSON()
}

func (c *AuthController) Callback() {
	code := c.GetString("code")
	state := c.GetString("state")

	if code == "" {
		c.CustomAbort(http.StatusBadRequest, "code es requerido")
		return
	}

	_ = state // si después quieres validar state

	resp, err := services.HandleCallback(&c.Controller, code)
	if err != nil {
		c.CustomAbort(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

func (c *AuthController) Logout() {
	resp, err := services.HandleLogout(&c.Controller)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

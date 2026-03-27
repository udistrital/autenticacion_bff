package controllers

import (
	"net/http"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/services"
)

type AuthController struct {
	beego.Controller
}

func (c *AuthController) Login() {
	clienteID := c.GetString("client_id")
	resp, err := services.HandleLogin(&c.Controller, clienteID)
	if err != nil {
		c.CustomAbort(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

func (c *AuthController) LoginURL() {
	clienteID := c.GetString("client_id")
	resp, err := services.HandleLogin(&c.Controller, clienteID)
	if err != nil {
		c.CustomAbort(http.StatusBadRequest, err.Error())
		return
	}

	c.Data["json"] = resp
	_ = c.ServeJSON()
}

func (c *AuthController) Callback() {
	code := c.GetString("code")
	state := c.GetString("state")

	resp, err := services.HandleCallback(&c.Controller, code, state)
	if err != nil {
		c.CustomAbort(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

func (c *AuthController) Logout() {
	clienteID := c.GetString("client_id")
	resp, err := services.HandleLogout(&c.Controller, clienteID)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

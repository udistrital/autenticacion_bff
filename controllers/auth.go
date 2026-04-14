package controllers

import (
	"net/http"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/services"
	"github.com/udistrital/autenticacion_bff/utils_oas/errorctrl"
)

// AuthController gestiona el flujo de autenticación del BFF contra Keycloak.
type AuthController struct {
	beego.Controller
}

// Login ...
// @Title Login
// @Description Inicia el flujo de autenticación contra Keycloak para el cliente indicado
// @Param	client_id		query 	string	true	"Identificador del cliente configurado en el BFF"
// @Success 307 {string} Redirección a Keycloak
// @Failure 400 client_id es inválido o no existe configuración para el cliente
// @router /login [get]
func (c *AuthController) Login() {
	defer errorctrl.ErrorControlController(c.Controller, "AuthController/Login")

	clienteID := c.GetString("client_id")
	resp, err := services.HandleLogin(&c.Controller, clienteID)
	if err != nil {
		panic(errorctrl.Error("Login - services.HandleLogin", err, "400"))
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

// LoginURL ...
// @Title LoginURL
// @Description Retorna la URL de autenticación contra Keycloak para el cliente indicado
// @Param	client_id		query 	string	true	"Identificador del cliente configurado en el BFF"
// @Success 200 {object} services.AuthRedirectResponse
// @Failure 400 client_id es inválido o no existe configuración para el cliente
// @router /login-url [get]
func (c *AuthController) LoginURL() {
	defer errorctrl.ErrorControlController(c.Controller, "AuthController/LoginURL")

	clienteID := c.GetString("client_id")
	resp, err := services.HandleLogin(&c.Controller, clienteID)
	if err != nil {
		panic(errorctrl.Error("LoginURL - services.HandleLogin", err, "400"))
	}

	c.Data["json"] = resp
	_ = c.ServeJSON()
}

// Callback ...
// @Title Callback
// @Description Procesa la respuesta de autenticación de Keycloak, valida el flujo y crea la sesión local
// @Param	code		query 	string	true	"Código de autorización retornado por Keycloak"
// @Param	state		query 	string	true	"Valor de estado usado para validar el flujo de autenticación"
// @Success 307 {string} Redirección al frontend configurado para el cliente
// @Failure 400 code o state inválidos, o error al procesar el callback
// @router /callback [get]
func (c *AuthController) Callback() {
	defer errorctrl.ErrorControlController(c.Controller, "AuthController/Callback")

	code := c.GetString("code")
	state := c.GetString("state")

	resp, err := services.HandleCallback(&c.Controller, code, state)
	if err != nil {
		panic(errorctrl.Error("Callback - services.HandleCallback", err, "400"))
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

// Logout ...
// @Title Logout
// @Description Cierra la sesión local del BFF y redirige al cierre de sesión configurado en Keycloak
// @Param	client_id		query 	string	false	"Identificador del cliente, usado como respaldo si no se puede resolver desde la sesión"
// @Success 307 {string} Redirección al endpoint de logout de Keycloak o al frontend configurado
// @Failure 500 error al cerrar la sesión
// @router /logout [get]
func (c *AuthController) Logout() {
	defer errorctrl.ErrorControlController(c.Controller, "AuthController/Logout")

	clienteID := c.GetString("client_id")
	resp, err := services.HandleLogout(&c.Controller, clienteID)
	if err != nil {
		panic(errorctrl.Error("Logout - services.HandleLogout", err, "500"))
	}

	c.Redirect(resp.RedirectURL, http.StatusTemporaryRedirect)
}

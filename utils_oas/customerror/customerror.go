package customerror

import (
	beego "github.com/beego/beego/v2/server/web"
)

// CustomErrorController centraliza la respuesta HTTP de errores del sistema.
// Su finalidad es asegurar que todos los errores respondan con una estructura JSON uniforme.
type CustomErrorController struct {
	beego.Controller
}

// genericError construye la estructura estándar de respuesta para los errores HTTP.
// Toma el código de estado recibido y utiliza los datos cargados previamente en c.Data
// para responder un JSON con información homogénea del error.
func genericError(c *CustomErrorController, status string) {
	outputError := map[string]interface{}{"Success": false, "Status": status, "Message": c.Data["mesaage"], "Data": c.Data["data"]}
	c.Data["json"] = outputError
	c.ServeJSON()
}

// Error400 Petición mala: indica que el servidor no puede o no procesará la petición debido a sintaxis inválida, tamaño de petición demasiado grande
func (c *CustomErrorController) Error400() {
	genericError(c, "400")
}

// Error401 No autorizado: indica que la petición (request) no ha sido ejecutada porque carece de credenciales válidas de autenticación
func (c *CustomErrorController) Error401() {
	genericError(c, "401")
}

// Error404 No encontrado: indica que el servidor no puede encontrar el recurso solicitado
func (c *CustomErrorController) Error404() {
	genericError(c, "404")
}

// Error500 Error interno de servidor: indica que el servidor encontró una condición inesperada que le impidió cumplir con la solicitud
func (c *CustomErrorController) Error500() {
	genericError(c, "500")
}

// Error501 No implementado: indica que el servidor no reconoce el método de solicitud porque no existe o esta mal llamada
func (c *CustomErrorController) Error501() {
	genericError(c, "501")
}

// Error502 Puerta de enlace no válida: indica que un servidor estaba actuando como puerta de enlace o proxy y que recibió una respuesta no válida del servidor ascendente
func (c *CustomErrorController) Error502() {
	genericError(c, "502")
}

// Error509 Límite de ancho de banda excedido: indica que el sitio web ha consumido todo el tráfico de datos asignado por su proveedor de alojamiento (hosting)
func (c *CustomErrorController) Error509() {
	genericError(c, "509")
}

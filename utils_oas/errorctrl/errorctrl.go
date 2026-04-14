package errorctrl

import (
	"net/http"
	"strconv"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

// ErrorControlController captura los panic generados dentro de un controlador,
// registra el error en logs, construye la trazabilidad del fallo y ejecuta el Abort
// con el código HTTP correspondiente para que sea procesado por el controlador de errores.
func ErrorControlController(c beego.Controller, controller string) {
	if err := recover(); err != nil {
		logs.Error(err)
		localError := err.(map[string]interface{})
		appName, _ := beego.AppConfig.String("appname")
		c.Data["mesaage"] = (appName + "/" + controller + "/" + (localError["funcion"]).(string))
		c.Data["data"] = (localError["err"])
		if status, ok := localError["status"]; ok && status != nil {
			c.Abort(status.(string))
		} else {
			c.Abort(strconv.Itoa(http.StatusInternalServerError)) // Unhandled Error!
		}
	}
}

// ErrorControlFunction captura un panic dentro de funciones internas o helpers
// y lo vuelve a propagar con la estructura estándar del sistema de errores.
// Su finalidad es mantener una única forma de escalar errores hacia el controlador.
func ErrorControlFunction(funcion string, status string) {
	if err := recover(); err != nil {
		panic(Error(funcion, err, status))
	}
}

// Error construye la estructura estándar de error utilizada en el sistema.
// Recibe el nombre de la función, el error original y el código HTTP asociado.
// Si el error ya viene en formato estándar, concatena la trazabilidad para no perder contexto.
func Error(funcion string, err interface{}, status string) (outputError map[string]interface{}) {
	switch localError := err.(type) {
	case map[string]interface{}:
		if fun, ok := localError["funcion"]; ok && fun != nil {
			funcion = funcion + "/" + fun.(string)
		}
		if er, ok := localError["err"]; ok && er != nil {
			err = er
		}
	}
	outputError = map[string]interface{}{"funcion": funcion, "err": err, "status": status}
	return outputError
}

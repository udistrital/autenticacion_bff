package apistatus

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func Init() {
	beego.Any("/", func(ctx *context.Context) {
		ctx.Output.JSON(map[string]interface{}{"status": "ok"}, true, true)
	})
}

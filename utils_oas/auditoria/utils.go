package auditoria

import (
	"encoding/json"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web/context"
)

func GetJsonWithHeader(urlp string, target interface{}, ctx *context.Context) error {
	req, err := http.NewRequest("GET", urlp, nil)
	if err != nil {
		logs.Error("Error reading request. ", err)
	}

	req.Header.Set("Authorization", ctx.Request.Header["Authorization"][0])
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logs.Error("Error reading response. ", err)
	}

	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

package request

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var global string

func SendJson(urlp string, trequest string, target interface{}, datajson interface{}) error {
	b := new(bytes.Buffer)
	if datajson != nil {
		json.NewEncoder(b).Encode(datajson)
	}
	//proxyUrl, err := url.Parse("http://10.20.4.15:3128")
	//http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}

	client := &http.Client{}
	req, err := http.NewRequest(trequest, urlp, b)
	if err != nil {
		logs.Error("Error creating request. ", err)
		return err
	}

	//Se intenta acceder a cabecera, si no existe, se realiza peticion normal.
	defer func() {
		//Catch
		if r := recover(); r != nil {

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				logs.Error("Error reading response. ", err)
			}

			defer resp.Body.Close()
			json.NewDecoder(resp.Body).Decode(target)
		}
	}()

	//try
	header := GetHeader()
	req.Header.Set("Authorization", header)

	resp, err := client.Do(req)
	if err != nil {
		logs.Error("Error reading response. ", err)
	}

	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func GetJsonWSO2(urlp string, target interface{}) error {
	b := new(bytes.Buffer)
	//proxyUrl, err := url.Parse("http://10.20.4.15:3128")
	//http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	client := &http.Client{}
	req, err := http.NewRequest("GET", urlp, b)
	if err != nil {
		logs.Error("Error creating request. ", err)
		return err
	}

	req.Header.Set("Accept", "application/json")
	r, err := client.Do(req)
	//r, err := http.Post(url, "application/json; charset=utf-8", b)
	if err != nil {
		logs.Error("error", err)
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func SetHeader(h string) {
	global = h

}

func GetHeader() (h string) {
	return global
}

func GetJson(urlp string, target interface{}) error {

	req, err := http.NewRequest("GET", urlp, nil)
	if err != nil {
		logs.Error("Error reading request. ", err)
	}

	//Se intenta acceder a cabecera, si no existe, se realiza peticion normal.

	defer func() {
		//Catch
		if r := recover(); r != nil {

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				logs.Error("Error reading response. ", err)
			}

			defer resp.Body.Close()
			json.NewDecoder(resp.Body).Decode(target)
		}
	}()

	//try
	header := GetHeader()
	req.Header.Set("Authorization", header)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logs.Error("Error reading response. ", err)
	}

	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func GetJsonTest(url string, target interface{}) (response *http.Response, err error) {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	response, err = myClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return response, json.NewDecoder(response.Body).Decode(target)
}

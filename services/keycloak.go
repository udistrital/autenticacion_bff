package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	beego "github.com/beego/beego/v2/server/web"
)

type KeycloakService struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

type UserInfo struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Email             string `json:"email"`
}

func NewKeycloakService() *KeycloakService {
	return &KeycloakService{
		BaseURL:      beego.AppConfig.DefaultString("keycloak_base_url", ""),
		Realm:        beego.AppConfig.DefaultString("keycloak_realm", ""),
		ClientID:     beego.AppConfig.DefaultString("keycloak_client_id", ""),
		ClientSecret: beego.AppConfig.DefaultString("keycloak_client_secret", ""),
		RedirectURI:  beego.AppConfig.DefaultString("keycloak_redirect_uri", ""),
	}
}

func (k *KeycloakService) RealmURL() string {
	return k.BaseURL + "/realms/" + k.Realm
}

func (k *KeycloakService) BuildLoginURL(state string) string {
	q := url.Values{}
	q.Set("client_id", k.ClientID)
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email")
	q.Set("redirect_uri", k.RedirectURI)
	q.Set("state", state)
	return k.RealmURL() + "/protocol/openid-connect/auth?" + q.Encode()
}

func (k *KeycloakService) BuildLogoutURL(idTokenHint, postLogoutRedirectURI string) string {
	q := url.Values{}
	if idTokenHint != "" {
		q.Set("id_token_hint", idTokenHint)
	}
	if postLogoutRedirectURI != "" {
		q.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}
	return k.RealmURL() + "/protocol/openid-connect/logout?" + q.Encode()
}

func (k *KeycloakService) ExchangeCode(code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", k.ClientID)
	form.Set("client_secret", k.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", k.RedirectURI)

	resp, err := http.Post(
		k.RealmURL()+"/protocol/openid-connect/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error intercambiando code: %s", string(body))
	}

	var out TokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (k *KeycloakService) RefreshTokens(refreshToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", k.ClientID)
	form.Set("client_secret", k.ClientSecret)
	form.Set("refresh_token", refreshToken)

	resp, err := http.Post(
		k.RealmURL()+"/protocol/openid-connect/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error refrescando tokens: %s", string(body))
	}

	var out TokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (k *KeycloakService) UserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", k.RealmURL()+"/protocol/openid-connect/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error consultando userinfo: %s", string(body))
	}

	var out UserInfo
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

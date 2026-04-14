package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// KeycloakService encapsula la configuración necesaria para interactuar
// con los endpoints OIDC de Keycloak.
type KeycloakService struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scope        string
}

// TokenResponse representa la respuesta estándar del endpoint de token de Keycloak.
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

// UserInfo representa la información básica del usuario retornada
// por el endpoint userinfo de Keycloak.
type UserInfo struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Email             string `json:"email"`
}

// NewKeycloakServiceForClient construye una instancia de servicio Keycloak
// a partir de la configuración del cliente y de la configuración global del sistema.
func NewKeycloakServiceForClient(cfg *ClientConfig) *KeycloakService {
	if cfg == nil {
		return &KeycloakService{}
	}

	return &KeycloakService{
		BaseURL:      strings.TrimSpace(beego.AppConfig.DefaultString("keycloak_base_url", "")),
		Realm:        strings.TrimSpace(beego.AppConfig.DefaultString("keycloak_realm", "")),
		ClientID:     strings.TrimSpace(cfg.ClienteID),
		ClientSecret: strings.TrimSpace(cfg.ClientSecret),
		RedirectURI:  strings.TrimSpace(cfg.RedirectURI),
		Scope:        strings.TrimSpace(cfg.Scope),
	}
}

// RealmURL construye la URL base del realm configurado.
func (k *KeycloakService) RealmURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(k.BaseURL), "/")
	realm := strings.Trim(strings.TrimSpace(k.Realm), "/")

	if baseURL == "" || realm == "" {
		return ""
	}

	return baseURL + "/realms/" + realm
}

// BuildLoginURL construye la URL de autenticación OIDC contra Keycloak.
// Requiere state y codeChallenge válidos para PKCE.
func (k *KeycloakService) BuildLoginURL(state string, codeChallenge string) string {
	if strings.TrimSpace(k.ClientID) == "" ||
		strings.TrimSpace(k.RedirectURI) == "" ||
		strings.TrimSpace(k.Scope) == "" ||
		strings.TrimSpace(state) == "" ||
		strings.TrimSpace(codeChallenge) == "" ||
		strings.TrimSpace(k.RealmURL()) == "" {
		return ""
	}

	q := url.Values{}
	q.Set("client_id", k.ClientID)
	q.Set("response_type", "code")
	q.Set("scope", k.Scope)
	q.Set("redirect_uri", k.RedirectURI)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")

	return k.RealmURL() + "/protocol/openid-connect/auth?" + q.Encode()
}

// BuildLogoutURL construye la URL de cierre de sesión contra Keycloak.
func (k *KeycloakService) BuildLogoutURL(idTokenHint, postLogoutRedirectURI string) string {
	if strings.TrimSpace(k.RealmURL()) == "" {
		return ""
	}

	q := url.Values{}
	if strings.TrimSpace(idTokenHint) != "" {
		q.Set("id_token_hint", idTokenHint)
	}
	if strings.TrimSpace(postLogoutRedirectURI) != "" {
		q.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	return k.RealmURL() + "/protocol/openid-connect/logout?" + q.Encode()
}

// ExchangeCode intercambia el authorization code por tokens en Keycloak.
func (k *KeycloakService) ExchangeCode(code string, codeVerifier string) (*TokenResponse, error) {
	if strings.TrimSpace(k.RealmURL()) == "" {
		return nil, fmt.Errorf("ExchangeCode: la URL del realm de Keycloak es inválida")
	}
	if strings.TrimSpace(k.ClientID) == "" {
		return nil, fmt.Errorf("ExchangeCode: client_id es requerido")
	}
	if strings.TrimSpace(k.ClientSecret) == "" {
		return nil, fmt.Errorf("ExchangeCode: client_secret es requerido")
	}
	if strings.TrimSpace(k.RedirectURI) == "" {
		return nil, fmt.Errorf("ExchangeCode: redirect_uri es requerido")
	}
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("ExchangeCode: code es requerido")
	}
	if strings.TrimSpace(codeVerifier) == "" {
		return nil, fmt.Errorf("ExchangeCode: code_verifier es requerido")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", k.ClientID)
	form.Set("client_secret", k.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", k.RedirectURI)
	form.Set("code_verifier", codeVerifier)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Post(
		k.RealmURL()+"/protocol/openid-connect/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("ExchangeCode: error consumiendo el endpoint de token: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("ExchangeCode: no fue posible leer la respuesta del endpoint de token: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ExchangeCode: error intercambiando code en Keycloak (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out TokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("ExchangeCode: no fue posible deserializar la respuesta de token: %w", err)
	}

	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("ExchangeCode: la respuesta no contiene access_token")
	}
	if strings.TrimSpace(out.TokenType) == "" {
		return nil, fmt.Errorf("ExchangeCode: la respuesta no contiene token_type")
	}

	return &out, nil
}

// RefreshTokens refresca el access_token usando el refresh_token actual.
func (k *KeycloakService) RefreshTokens(refreshToken string) (*TokenResponse, error) {
	if strings.TrimSpace(k.RealmURL()) == "" {
		return nil, fmt.Errorf("RefreshTokens: la URL del realm de Keycloak es inválida")
	}
	if strings.TrimSpace(k.ClientID) == "" {
		return nil, fmt.Errorf("RefreshTokens: client_id es requerido")
	}
	if strings.TrimSpace(k.ClientSecret) == "" {
		return nil, fmt.Errorf("RefreshTokens: client_secret es requerido")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("RefreshTokens: refresh_token es requerido")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", k.ClientID)
	form.Set("client_secret", k.ClientSecret)
	form.Set("refresh_token", refreshToken)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Post(
		k.RealmURL()+"/protocol/openid-connect/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("RefreshTokens: error consumiendo el endpoint de token: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("RefreshTokens: no fue posible leer la respuesta del endpoint de token: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("RefreshTokens: error refrescando tokens en Keycloak (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out TokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("RefreshTokens: no fue posible deserializar la respuesta de token: %w", err)
	}

	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("RefreshTokens: la respuesta no contiene access_token")
	}
	if strings.TrimSpace(out.TokenType) == "" {
		return nil, fmt.Errorf("RefreshTokens: la respuesta no contiene token_type")
	}

	return &out, nil
}

// UserInfo consulta la información básica del usuario usando el access_token actual.
func (k *KeycloakService) UserInfo(accessToken string) (*UserInfo, error) {
	if strings.TrimSpace(k.RealmURL()) == "" {
		return nil, fmt.Errorf("UserInfo: la URL del realm de Keycloak es inválida")
	}
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("UserInfo: access_token es requerido")
	}

	req, err := http.NewRequest("GET", k.RealmURL()+"/protocol/openid-connect/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("UserInfo: no fue posible construir la petición al endpoint userinfo: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("UserInfo: error consumiendo el endpoint userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("UserInfo: no fue posible leer la respuesta del endpoint userinfo: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("UserInfo: error consultando userinfo en Keycloak (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out UserInfo
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("UserInfo: no fue posible deserializar la respuesta de userinfo: %w", err)
	}

	if strings.TrimSpace(out.Sub) == "" {
		return nil, fmt.Errorf("UserInfo: la respuesta no contiene sub")
	}

	return &out, nil
}

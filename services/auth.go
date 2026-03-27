package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

type AuthRedirectResponse struct {
	RedirectURL string `json:"redirect_url"`
}

type loginContext struct {
	ClienteID    string `json:"client_id"`
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
	CreatedAt    int64  `json:"created_at"`
}

func HandleLogin(c *beego.Controller, clienteID string) (*AuthRedirectResponse, error) {
	cfg, err := GetClientConfig(clienteID)
	if err != nil {
		return nil, err
	}

	state, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	codeVerifier, err := generateRandomString(64)
	if err != nil {
		return nil, err
	}

	codeChallenge := buildS256CodeChallenge(codeVerifier)

	ctx := loginContext{
		ClienteID:    cfg.ClienteID,
		State:        state,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now().UTC().Unix(),
	}

	if err := setLoginContextCookie(c, ctx); err != nil {
		return nil, err
	}

	keycloak := NewKeycloakServiceForClient(cfg)
	authURL := keycloak.BuildLoginURL(state, codeChallenge)

	return &AuthRedirectResponse{
		RedirectURL: authURL,
	}, nil
}

func HandleCallback(c *beego.Controller, code string, state string) (*AuthRedirectResponse, error) {
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("code es requerido")
	}
	if strings.TrimSpace(state) == "" {
		return nil, errors.New("state es requerido")
	}

	ctx, err := getLoginContextCookie(c)
	if err != nil {
		return nil, err
	}

	if ctx.State != state {
		return nil, errors.New("state inválido")
	}

	if time.Now().UTC().Unix()-ctx.CreatedAt > 300 {
		return nil, errors.New("contexto de login expirado")
	}

	cfg, err := GetClientConfig(ctx.ClienteID)
	if err != nil {
		return nil, err
	}

	keycloak := NewKeycloakServiceForClient(cfg)

	tokenData, err := keycloak.ExchangeCode(code, ctx.CodeVerifier)
	if err != nil {
		return nil, err
	}

	userInfo, err := keycloak.UserInfo(tokenData.AccessToken)
	if err != nil {
		return nil, err
	}

	sessionService, err := NewSessionService()
	if err != nil {
		return nil, err
	}

	var ip *string
	ipValue := clientIP(c.Ctx.Request)
	if ipValue != "" {
		ip = &ipValue
	}

	var userAgent *string
	ua := c.Ctx.Request.UserAgent()
	if ua != "" {
		userAgent = &ua
	}

	var nombreUsuario *string
	if userInfo.Name != "" {
		nombreUsuario = &userInfo.Name
	} else if userInfo.PreferredUsername != "" {
		nombreUsuario = &userInfo.PreferredUsername
	}

	var correoElectronico *string
	if userInfo.Email != "" {
		correoElectronico = &userInfo.Email
	}

	var refreshToken *string
	if tokenData.RefreshToken != "" {
		refreshToken = &tokenData.RefreshToken
	}

	var idToken *string
	if tokenData.IDToken != "" {
		idToken = &tokenData.IDToken
	}

	expiresIn := tokenData.ExpiresIn
	refreshExpiresIn := tokenData.RefreshExpiresIn

	idSesion, err := sessionService.CrearSesion(
		cfg.ClienteID,
		userInfo.Sub,
		nombreUsuario,
		correoElectronico,
		tokenData.AccessToken,
		refreshToken,
		idToken,
		tokenData.TokenType,
		&expiresIn,
		&refreshExpiresIn,
		ip,
		userAgent,
	)
	if err != nil {
		return nil, err
	}

	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{
		Name:     beego.AppConfig.DefaultString("session_cookie_name", "id_sesion"),
		Value:    idSesion,
		Path:     beego.AppConfig.DefaultString("session_cookie_path", "/"),
		HttpOnly: true,
		Secure:   beego.AppConfig.DefaultBool("session_cookie_secure", false),
		MaxAge:   4 * 60 * 60,
		SameSite: parseSameSite(beego.AppConfig.DefaultString("session_cookie_samesite", "Lax")),
	})

	clearLoginContextCookie(c)

	return &AuthRedirectResponse{
		RedirectURL: cfg.FrontendSuccessRedirect,
	}, nil
}

func HandleLogout(c *beego.Controller, clienteIDParam string) (*AuthRedirectResponse, error) {
	cookieName := beego.AppConfig.DefaultString("session_cookie_name", "id_sesion")
	cookie, err := c.Ctx.Request.Cookie(cookieName)

	sessionService, serr := NewSessionService()
	if serr != nil {
		return nil, serr
	}

	var idToken string
	var clienteID string

	if err == nil && cookie != nil && cookie.Value != "" {
		sesion, _ := sessionService.ObtenerSesionPorID(cookie.Value)
		if sesion != nil {
			clienteID = sesion.ClienteID
			if sesion.IDToken != nil {
				idToken = *sesion.IDToken
			}
		}
		_ = sessionService.RevocarSesion(cookie.Value)
	}

	if clienteID == "" {
		clienteID = strings.TrimSpace(clienteIDParam)
	}

	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     beego.AppConfig.DefaultString("session_cookie_path", "/"),
		HttpOnly: true,
		Secure:   beego.AppConfig.DefaultBool("session_cookie_secure", false),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		SameSite: parseSameSite(beego.AppConfig.DefaultString("session_cookie_samesite", "Lax")),
	})

	clearLoginContextCookie(c)

	if clienteID == "" {
		return nil, errors.New("no fue posible resolver el client_id para logout")
	}

	cfg, err := GetClientConfig(clienteID)
	if err != nil {
		return nil, err
	}

	if idToken == "" {
		return &AuthRedirectResponse{
			RedirectURL: cfg.FrontendLogoutRedirect,
		}, nil
	}

	keycloak := NewKeycloakServiceForClient(cfg)
	logoutURL := keycloak.BuildLogoutURL(idToken, cfg.FrontendLogoutRedirect)

	return &AuthRedirectResponse{
		RedirectURL: logoutURL,
	}, nil
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	if r.RemoteAddr != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return ""
}

func ValidateCode(code string) error {
	if strings.TrimSpace(code) == "" {
		return errors.New("code es requerido")
	}
	return nil
}

func generateRandomString(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("error generando valor aleatorio: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func buildS256CodeChallenge(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func signPayload(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func getLoginSigningKey() (string, error) {
	key := beego.AppConfig.DefaultString("login_context_signing_key", "")
	if strings.TrimSpace(key) == "" {
		return "", errors.New("login_context_signing_key no está configurado")
	}
	return key, nil
}

func setLoginContextCookie(c *beego.Controller, ctx loginContext) error {
	key, err := getLoginSigningKey()
	if err != nil {
		return err
	}

	raw, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("error serializando login context: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(raw)
	signature := signPayload(payload, key)
	value := payload + "." + signature

	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{
		Name:     beego.AppConfig.DefaultString("login_context_cookie_name", "kc_login_ctx"),
		Value:    value,
		Path:     beego.AppConfig.DefaultString("login_context_cookie_path", "/"),
		HttpOnly: true,
		Secure:   beego.AppConfig.DefaultBool("login_context_cookie_secure", false),
		MaxAge:   300,
		SameSite: parseSameSite(beego.AppConfig.DefaultString("login_context_cookie_samesite", "Lax")),
	})

	return nil
}

func getLoginContextCookie(c *beego.Controller) (*loginContext, error) {
	key, err := getLoginSigningKey()
	if err != nil {
		return nil, err
	}

	cookieName := beego.AppConfig.DefaultString("login_context_cookie_name", "kc_login_ctx")
	cookie, err := c.Ctx.Request.Cookie(cookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil, errors.New("contexto de login no encontrado")
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return nil, errors.New("contexto de login inválido")
	}

	payload := parts[0]
	signature := parts[1]
	expected := signPayload(payload, key)

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return nil, errors.New("firma de contexto de login inválida")
	}

	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("payload de contexto de login inválido")
	}

	var ctx loginContext
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return nil, errors.New("no fue posible leer el contexto de login")
	}

	return &ctx, nil
}

func clearLoginContextCookie(c *beego.Controller) {
	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{
		Name:     beego.AppConfig.DefaultString("login_context_cookie_name", "kc_login_ctx"),
		Value:    "",
		Path:     beego.AppConfig.DefaultString("login_context_cookie_path", "/"),
		HttpOnly: true,
		Secure:   beego.AppConfig.DefaultBool("login_context_cookie_secure", false),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		SameSite: parseSameSite(beego.AppConfig.DefaultString("login_context_cookie_samesite", "Lax")),
	})
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

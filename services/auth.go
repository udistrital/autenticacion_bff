package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

// HandleLogin construye la URL de autenticación para el cliente indicado
// y almacena el contexto temporal del flujo OIDC en cookie firmada.
func HandleLogin(c *beego.Controller, clienteID string) (*AuthRedirectResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("HandleLogin: el controlador es requerido")
	}

	clienteID = strings.TrimSpace(clienteID)
	if clienteID == "" {
		return nil, fmt.Errorf("HandleLogin: client_id es requerido")
	}

	cfg, err := GetClientConfig(clienteID)
	if err != nil {
		return nil, fmt.Errorf("HandleLogin: no fue posible obtener la configuración del cliente %s: %w", clienteID, err)
	}

	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("HandleLogin: no fue posible generar el state: %w", err)
	}

	codeVerifier, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("HandleLogin: no fue posible generar el code_verifier: %w", err)
	}

	codeChallenge := buildS256CodeChallenge(codeVerifier)

	ctx := loginContext{
		ClienteID:    cfg.ClienteID,
		State:        state,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now().UTC().Unix(),
	}

	if err := setLoginContextCookie(c, ctx); err != nil {
		return nil, fmt.Errorf("HandleLogin: no fue posible almacenar el contexto de login: %w", err)
	}

	keycloak := NewKeycloakServiceForClient(cfg)
	authURL := keycloak.BuildLoginURL(state, codeChallenge)
	if strings.TrimSpace(authURL) == "" {
		return nil, fmt.Errorf("HandleLogin: no fue posible construir la URL de autenticación")
	}

	return &AuthRedirectResponse{
		RedirectURL: authURL,
	}, nil
}

// HandleCallback procesa la respuesta de Keycloak, valida el contexto del login
// y crea la sesión local del usuario autenticado.
func HandleCallback(c *beego.Controller, code string, state string) (*AuthRedirectResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("HandleCallback: el controlador es requerido")
	}

	code = strings.TrimSpace(code)
	state = strings.TrimSpace(state)

	if code == "" {
		return nil, fmt.Errorf("HandleCallback: code es requerido")
	}
	if state == "" {
		return nil, fmt.Errorf("HandleCallback: state es requerido")
	}

	ctx, err := getLoginContextCookie(c)
	if err != nil {
		return nil, fmt.Errorf("HandleCallback: no fue posible obtener el contexto de login: %w", err)
	}
	if ctx == nil {
		return nil, fmt.Errorf("HandleCallback: el contexto de login es inválido")
	}

	if strings.TrimSpace(ctx.State) == "" {
		return nil, fmt.Errorf("HandleCallback: el contexto de login no contiene state")
	}
	if ctx.State != state {
		return nil, fmt.Errorf("HandleCallback: state inválido")
	}

	if time.Now().UTC().Unix()-ctx.CreatedAt > 300 {
		return nil, fmt.Errorf("HandleCallback: el contexto de login expiró")
	}

	if strings.TrimSpace(ctx.ClienteID) == "" {
		return nil, fmt.Errorf("HandleCallback: el contexto de login no contiene client_id")
	}

	cfg, err := GetClientConfig(ctx.ClienteID)
	if err != nil {
		return nil, fmt.Errorf("HandleCallback: no fue posible obtener la configuración del cliente %s: %w", ctx.ClienteID, err)
	}

	keycloak := NewKeycloakServiceForClient(cfg)

	tokenData, err := keycloak.ExchangeCode(code, ctx.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("HandleCallback: no fue posible intercambiar el code por tokens: %w", err)
	}
	if tokenData == nil {
		return nil, fmt.Errorf("HandleCallback: Keycloak no retornó información de tokens")
	}
	if strings.TrimSpace(tokenData.AccessToken) == "" {
		return nil, fmt.Errorf("HandleCallback: el access_token retornado por Keycloak es inválido")
	}
	if strings.TrimSpace(tokenData.TokenType) == "" {
		return nil, fmt.Errorf("HandleCallback: el token_type retornado por Keycloak es inválido")
	}

	userInfo, err := keycloak.UserInfo(tokenData.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("HandleCallback: no fue posible consultar la información del usuario: %w", err)
	}
	if userInfo == nil {
		return nil, fmt.Errorf("HandleCallback: Keycloak no retornó información del usuario")
	}
	if strings.TrimSpace(userInfo.Sub) == "" {
		return nil, fmt.Errorf("HandleCallback: la información del usuario no contiene sub")
	}

	sessionService, err := NewSessionService()
	if err != nil {
		return nil, fmt.Errorf("HandleCallback: no fue posible inicializar el servicio de sesión: %w", err)
	}

	var ip *string
	ipValue := clientIP(c.Ctx.Request)
	if ipValue != "" {
		ip = &ipValue
	}

	var userAgent *string
	ua := strings.TrimSpace(c.Ctx.Request.UserAgent())
	if ua != "" {
		userAgent = &ua
	}

	var nombreUsuario *string
	if strings.TrimSpace(userInfo.Name) != "" {
		name := strings.TrimSpace(userInfo.Name)
		nombreUsuario = &name
	} else if strings.TrimSpace(userInfo.PreferredUsername) != "" {
		username := strings.TrimSpace(userInfo.PreferredUsername)
		nombreUsuario = &username
	}

	var correoElectronico *string
	if strings.TrimSpace(userInfo.Email) != "" {
		email := strings.TrimSpace(userInfo.Email)
		correoElectronico = &email
	}

	var refreshToken *string
	if strings.TrimSpace(tokenData.RefreshToken) != "" {
		rt := strings.TrimSpace(tokenData.RefreshToken)
		refreshToken = &rt
	}

	var idToken *string
	if strings.TrimSpace(tokenData.IDToken) != "" {
		it := strings.TrimSpace(tokenData.IDToken)
		idToken = &it
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
		return nil, fmt.Errorf("HandleCallback: no fue posible crear la sesión del usuario: %w", err)
	}
	if strings.TrimSpace(idSesion) == "" {
		return nil, fmt.Errorf("HandleCallback: no fue posible generar el identificador de sesión")
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

// HandleLogout revoca la sesión local y construye la redirección de logout
// de Keycloak o del frontend, según la información disponible.
func HandleLogout(c *beego.Controller, clienteIDParam string) (*AuthRedirectResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("HandleLogout: el controlador es requerido")
	}

	cookieName := beego.AppConfig.DefaultString("session_cookie_name", "id_sesion")
	cookie, err := c.Ctx.Request.Cookie(cookieName)

	sessionService, serr := NewSessionService()
	if serr != nil {
		return nil, fmt.Errorf("HandleLogout: no fue posible inicializar el servicio de sesión: %w", serr)
	}

	var idToken string
	var clienteID string

	if err == nil && cookie != nil && strings.TrimSpace(cookie.Value) != "" {
		sesion, sessionErr := sessionService.ObtenerSesionPorID(cookie.Value)
		if sessionErr != nil {
			return nil, fmt.Errorf("HandleLogout: no fue posible consultar la sesión actual: %w", sessionErr)
		}

		if sesion != nil {
			clienteID = strings.TrimSpace(sesion.ClienteID)
			if sesion.IDToken != nil {
				idToken = strings.TrimSpace(*sesion.IDToken)
			}
		}

		if revokeErr := sessionService.RevocarSesion(cookie.Value); revokeErr != nil {
			return nil, fmt.Errorf("HandleLogout: no fue posible revocar la sesión: %w", revokeErr)
		}
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
		return nil, fmt.Errorf("HandleLogout: no fue posible resolver el client_id para logout")
	}

	cfg, err := GetClientConfig(clienteID)
	if err != nil {
		return nil, fmt.Errorf("HandleLogout: no fue posible obtener la configuración del cliente %s: %w", clienteID, err)
	}

	if strings.TrimSpace(idToken) == "" {
		return &AuthRedirectResponse{
			RedirectURL: cfg.FrontendLogoutRedirect,
		}, nil
	}

	keycloak := NewKeycloakServiceForClient(cfg)
	logoutURL := keycloak.BuildLogoutURL(idToken, cfg.FrontendLogoutRedirect)
	if strings.TrimSpace(logoutURL) == "" {
		return nil, fmt.Errorf("HandleLogout: no fue posible construir la URL de logout")
	}

	return &AuthRedirectResponse{
		RedirectURL: logoutURL,
	}, nil
}

// clientIP obtiene la dirección IP del cliente priorizando cabeceras de proxy
// y usando RemoteAddr como último respaldo.
func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}

	if r.RemoteAddr != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return ""
}

// ValidateCode valida que el code de autorización recibido en el callback
// no llegue vacío.
func ValidateCode(code string) error {
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("ValidateCode: code es requerido")
	}
	return nil
}

// generateRandomString genera una cadena aleatoria segura en formato base64url
// para su uso en state y code_verifier.
func generateRandomString(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("generateRandomString: el tamaño debe ser mayor que cero")
	}

	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generateRandomString: error generando valor aleatorio: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// buildS256CodeChallenge construye el code_challenge en formato S256
// a partir del code_verifier.
func buildS256CodeChallenge(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// signPayload firma el payload usando HMAC-SHA256 y retorna el valor
// en base64url.
func signPayload(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// getLoginSigningKey obtiene la clave usada para firmar la cookie temporal
// del contexto de login.
func getLoginSigningKey() (string, error) {
	key := strings.TrimSpace(beego.AppConfig.DefaultString("login_context_signing_key", ""))
	if key == "" {
		return "", fmt.Errorf("getLoginSigningKey: login_context_signing_key no está configurado")
	}
	return key, nil
}

// setLoginContextCookie serializa, firma y almacena en cookie el contexto temporal
// requerido para completar el flujo OIDC con PKCE.
func setLoginContextCookie(c *beego.Controller, ctx loginContext) error {
	if c == nil {
		return fmt.Errorf("setLoginContextCookie: el controlador es requerido")
	}
	if strings.TrimSpace(ctx.ClienteID) == "" {
		return fmt.Errorf("setLoginContextCookie: client_id es requerido")
	}
	if strings.TrimSpace(ctx.State) == "" {
		return fmt.Errorf("setLoginContextCookie: state es requerido")
	}
	if strings.TrimSpace(ctx.CodeVerifier) == "" {
		return fmt.Errorf("setLoginContextCookie: code_verifier es requerido")
	}

	key, err := getLoginSigningKey()
	if err != nil {
		return fmt.Errorf("setLoginContextCookie: %w", err)
	}

	raw, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("setLoginContextCookie: error serializando login context: %w", err)
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

// getLoginContextCookie recupera y valida la cookie temporal del flujo de login,
// verificando firma, estructura y contenido.
func getLoginContextCookie(c *beego.Controller) (*loginContext, error) {
	if c == nil {
		return nil, fmt.Errorf("getLoginContextCookie: el controlador es requerido")
	}

	key, err := getLoginSigningKey()
	if err != nil {
		return nil, fmt.Errorf("getLoginContextCookie: %w", err)
	}

	cookieName := beego.AppConfig.DefaultString("login_context_cookie_name", "kc_login_ctx")
	cookie, err := c.Ctx.Request.Cookie(cookieName)
	if err != nil || cookie == nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, fmt.Errorf("getLoginContextCookie: contexto de login no encontrado")
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("getLoginContextCookie: contexto de login inválido")
	}

	payload := parts[0]
	signature := parts[1]
	expected := signPayload(payload, key)

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return nil, fmt.Errorf("getLoginContextCookie: firma de contexto de login inválida")
	}

	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("getLoginContextCookie: payload de contexto de login inválido: %w", err)
	}

	var ctx loginContext
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return nil, fmt.Errorf("getLoginContextCookie: no fue posible leer el contexto de login: %w", err)
	}

	if strings.TrimSpace(ctx.ClienteID) == "" {
		return nil, fmt.Errorf("getLoginContextCookie: el contexto de login no contiene client_id")
	}
	if strings.TrimSpace(ctx.State) == "" {
		return nil, fmt.Errorf("getLoginContextCookie: el contexto de login no contiene state")
	}
	if strings.TrimSpace(ctx.CodeVerifier) == "" {
		return nil, fmt.Errorf("getLoginContextCookie: el contexto de login no contiene code_verifier")
	}

	return &ctx, nil
}

// clearLoginContextCookie elimina la cookie temporal del flujo de login.
func clearLoginContextCookie(c *beego.Controller) {
	if c == nil {
		return
	}

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

// parseSameSite traduce el valor configurado en app.conf al tipo http.SameSite
// esperado por la librería estándar.
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

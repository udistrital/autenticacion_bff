package services

import (
	"errors"
	"net/http"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/utils_oas/errorctrl"
)

type AuthRedirectResponse struct {
	RedirectURL string
}

func HandleCallback(c *beego.Controller, code string) (*AuthRedirectResponse, error) {
	keycloak := NewKeycloakService()

	tokenData, err := keycloak.ExchangeCode(code)
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
		Path:     "/",
		HttpOnly: true,
		Secure:   beego.AppConfig.DefaultBool("session_cookie_secure", false),
		MaxAge:   4 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURL, err := beego.AppConfig.String("frontend_success_redirect")
	if err != nil || redirectURL == "" {
		panic(errorctrl.Error("HandleCallback", "frontend_success_redirect no está configurado", "500"))
	}

	return &AuthRedirectResponse{
		RedirectURL: redirectURL,
	}, nil
}

func HandleLogout(c *beego.Controller) (*AuthRedirectResponse, error) {
	cookieName := beego.AppConfig.DefaultString("session_cookie_name", "id_sesion")
	cookie, err := c.Ctx.Request.Cookie(cookieName)

	sessionService, serr := NewSessionService()
	if serr != nil {
		return nil, serr
	}

	var idToken string

	if err == nil && cookie != nil && cookie.Value != "" {
		sesion, _ := sessionService.ObtenerSesionPorID(cookie.Value)
		if sesion != nil && sesion.IDToken != nil {
			idToken = *sesion.IDToken
		}
		_ = sessionService.RevocarSesion(cookie.Value)
	}

	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})

	postLogoutRedirect, err := beego.AppConfig.String("frontend_logout_redirect")
	if err != nil || postLogoutRedirect == "" {
		panic(errorctrl.Error("HandleCallback", "frontend_logout_redirect no está configurado", "500"))
	}

	if idToken == "" {
		return &AuthRedirectResponse{
			RedirectURL: postLogoutRedirect,
		}, nil
	}

	keycloak := NewKeycloakService()
	logoutURL := keycloak.BuildLogoutURL(idToken, postLogoutRedirect)

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

	host := r.RemoteAddr
	if host == "" {
		return ""
	}

	return host
}

func ValidateCode(code string) error {
	if strings.TrimSpace(code) == "" {
		return errors.New("code es requerido")
	}
	return nil
}

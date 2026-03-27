package services

import (
	"database/sql"
	"fmt"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
)

type ClientConfig struct {
	ClienteID               string
	ClientSecret            string
	RedirectURI             string
	FrontendSuccessRedirect string
	FrontendLogoutRedirect  string
	Scope                   string
	Activo                  bool
}

func GetClientConfig(clienteID string) (*ClientConfig, error) {
	clienteID = strings.TrimSpace(clienteID)
	if clienteID == "" {
		return nil, fmt.Errorf("client_id es requerido")
	}

	db, err := GetPostgresDB()
	if err != nil {
		return nil, err
	}

	schema := beego.AppConfig.DefaultString("postgres_schema", "configuracion")
	query := fmt.Sprintf(`
		SELECT
			cliente_id,
			client_secret,
			redirect_uri,
			frontend_success_redirect,
			frontend_logout_redirect,
			COALESCE(NULLIF(scope, ''), $2) AS scope,
			activo
		FROM %s.clientes
		WHERE cliente_id = $1
		LIMIT 1
	`, schema)

	defaultScope := beego.AppConfig.DefaultString("keycloak_default_scope", "openid profile email")

	var cfg ClientConfig
	err = db.QueryRow(query, clienteID, defaultScope).Scan(
		&cfg.ClienteID,
		&cfg.ClientSecret,
		&cfg.RedirectURI,
		&cfg.FrontendSuccessRedirect,
		&cfg.FrontendLogoutRedirect,
		&cfg.Scope,
		&cfg.Activo,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no existe configuración para el client_id %s", clienteID)
		}
		return nil, fmt.Errorf("error consultando configuración del cliente %s: %w", clienteID, err)
	}

	if !cfg.Activo {
		return nil, fmt.Errorf("el cliente %s está inactivo", clienteID)
	}

	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, fmt.Errorf("el cliente %s no tiene client_secret configurado", clienteID)
	}
	if strings.TrimSpace(cfg.RedirectURI) == "" {
		return nil, fmt.Errorf("el cliente %s no tiene redirect_uri configurado", clienteID)
	}
	if strings.TrimSpace(cfg.FrontendSuccessRedirect) == "" {
		return nil, fmt.Errorf("el cliente %s no tiene frontend_success_redirect configurado", clienteID)
	}
	if strings.TrimSpace(cfg.FrontendLogoutRedirect) == "" {
		return nil, fmt.Errorf("el cliente %s no tiene frontend_logout_redirect configurado", clienteID)
	}
	if strings.TrimSpace(cfg.Scope) == "" {
		cfg.Scope = defaultScope
	}

	return &cfg, nil
}

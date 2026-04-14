package services

import (
	"database/sql"
	"fmt"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
)

// ClientConfig representa la configuración OIDC de un cliente registrada en base de datos.
type ClientConfig struct {
	ClienteID               string
	ClientSecret            string
	RedirectURI             string
	FrontendSuccessRedirect string
	FrontendLogoutRedirect  string
	Scope                   string
	Activo                  bool
}

// GetClientConfig consulta la configuración OIDC de un cliente a partir de su client_id.
// Valida que el cliente exista, esté activo y tenga todos los campos mínimos requeridos
// para construir el flujo de autenticación contra Keycloak.
func GetClientConfig(clienteID string) (*ClientConfig, error) {
	clienteID = strings.TrimSpace(clienteID)
	if clienteID == "" {
		return nil, fmt.Errorf("GetClientConfig: client_id es requerido")
	}

	db, err := GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("GetClientConfig: no fue posible obtener la conexión a PostgreSQL: %w", err)
	}
	if db == nil {
		return nil, fmt.Errorf("GetClientConfig: la conexión a PostgreSQL es inválida")
	}

	schema := strings.TrimSpace(beego.AppConfig.DefaultString("postgres_schema", "configuracion"))
	if schema == "" {
		schema = "configuracion"
	}

	defaultScope := strings.TrimSpace(beego.AppConfig.DefaultString("keycloak_default_scope", "openid profile email"))
	if defaultScope == "" {
		defaultScope = "openid profile email"
	}

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
			return nil, fmt.Errorf("GetClientConfig: no existe configuración para el client_id %s", clienteID)
		}
		return nil, fmt.Errorf("GetClientConfig: error consultando configuración del cliente %s: %w", clienteID, err)
	}

	cfg.ClienteID = strings.TrimSpace(cfg.ClienteID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.RedirectURI = strings.TrimSpace(cfg.RedirectURI)
	cfg.FrontendSuccessRedirect = strings.TrimSpace(cfg.FrontendSuccessRedirect)
	cfg.FrontendLogoutRedirect = strings.TrimSpace(cfg.FrontendLogoutRedirect)
	cfg.Scope = strings.TrimSpace(cfg.Scope)

	if cfg.ClienteID == "" {
		return nil, fmt.Errorf("GetClientConfig: la configuración del cliente %s no contiene cliente_id válido", clienteID)
	}

	if !cfg.Activo {
		return nil, fmt.Errorf("GetClientConfig: el cliente %s está inactivo", clienteID)
	}

	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("GetClientConfig: el cliente %s no tiene client_secret configurado", clienteID)
	}

	if cfg.RedirectURI == "" {
		return nil, fmt.Errorf("GetClientConfig: el cliente %s no tiene redirect_uri configurado", clienteID)
	}

	if cfg.FrontendSuccessRedirect == "" {
		return nil, fmt.Errorf("GetClientConfig: el cliente %s no tiene frontend_success_redirect configurado", clienteID)
	}

	if cfg.FrontendLogoutRedirect == "" {
		return nil, fmt.Errorf("GetClientConfig: el cliente %s no tiene frontend_logout_redirect configurado", clienteID)
	}

	if cfg.Scope == "" {
		cfg.Scope = defaultScope
	}

	return &cfg, nil
}

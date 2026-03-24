package models

type Session struct {
	IDSesion                     string  `dynamodbav:"id_sesion" json:"id_sesion"`
	IDUsuario                    string  `dynamodbav:"id_usuario" json:"id_usuario"`
	NombreUsuario                *string `dynamodbav:"nombre_usuario" json:"nombre_usuario"`
	CorreoElectronico            *string `dynamodbav:"correo_electronico" json:"correo_electronico"`
	TokenAcceso                  *string `dynamodbav:"token_acceso" json:"token_acceso"`
	TokenActualizacion           *string `dynamodbav:"token_actualizacion" json:"token_actualizacion"`
	IDToken                      *string `dynamodbav:"id_token" json:"id_token"`
	FechaExpiracionToken         string  `dynamodbav:"fecha_expiracion_token" json:"fecha_expiracion_token"`
	TipoToken                    string  `dynamodbav:"tipo_token" json:"tipo_token"`
	FechaExpiracion              string  `dynamodbav:"fecha_expiracion" json:"fecha_expiracion"`
	FechaExpiracionActualizacion *string `dynamodbav:"fecha_expiracion_actualizacion" json:"fecha_expiracion_actualizacion"`
	FechaCreacion                string  `dynamodbav:"fecha_creacion" json:"fecha_creacion"`
	FechaUltimaActividad         string  `dynamodbav:"fecha_ultima_actividad" json:"fecha_ultima_actividad"`
	FechaRevocacion              *string `dynamodbav:"fecha_revocacion" json:"fecha_revocacion"`
	Activo                       bool    `dynamodbav:"activo" json:"activo"`
	DireccionIP                  *string `dynamodbav:"direccion_ip" json:"direccion_ip"`
	AgenteUsuario                *string `dynamodbav:"agente_usuario" json:"agente_usuario"`
	TTL                          int64   `dynamodbav:"ttl" json:"ttl"`
}

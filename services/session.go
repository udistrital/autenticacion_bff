package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/autenticacion_bff/models"
)

type SessionService struct {
	client    *dynamodb.Client
	tableName string
}

func NewSessionService() (*SessionService, error) {
	region := web.AppConfig.DefaultString("aws_region", "us-east-1")
	tableName := web.AppConfig.DefaultString("dynamodb_table_sesiones", "sesiones")
	endpointURL := web.AppConfig.DefaultString("dynamodb_endpoint_url", "")
	accessKey := web.AppConfig.DefaultString("aws_access_key_id", "")
	secretKey := web.AppConfig.DefaultString("aws_secret_access_key", "")

	var (
		cfg aws.Config
		err error
	)

	if accessKey != "" && secretKey != "" {
		cfg, err = awsconfig.LoadDefaultConfig(
			context.TODO(),
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(
				aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
					return aws.Credentials{
						AccessKeyID:     accessKey,
						SecretAccessKey: secretKey,
						Source:          "static",
					}, nil
				}),
			),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(
			context.TODO(),
			awsconfig.WithRegion(region),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("error cargando configuración AWS: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpointURL != "" {
			o.BaseEndpoint = aws.String(endpointURL)
		}
	})

	return &SessionService{
		client:    client,
		tableName: tableName,
	}, nil
}

func (s *SessionService) now() time.Time {
	return time.Now().UTC()
}

func (s *SessionService) toISO(t *time.Time) *string {
	if t == nil {
		return nil
	}
	value := t.UTC().Format(time.RFC3339)
	return &value
}

func (s *SessionService) generateSessionID() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("error generando id de sesión: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *SessionService) CrearSesion(
	clienteID string,
	idUsuario string,
	nombreUsuario *string,
	correoElectronico *string,
	tokenAcceso string,
	tokenActualizacion *string,
	idToken *string,
	tipoToken string,
	expiraEnSegundos *int,
	expiraActualizacionEnSegundos *int,
	direccionIP *string,
	agenteUsuario *string,
) (string, error) {
	idSesion, err := s.generateSessionID()
	if err != nil {
		return "", err
	}

	ahora := s.now()

	expToken := ahora.Add(3600 * time.Second)
	if expiraEnSegundos != nil {
		expToken = ahora.Add(time.Duration(*expiraEnSegundos) * time.Second)
	}

	expSesion := ahora.Add(4 * time.Hour)

	var expRefresh *time.Time
	if expiraActualizacionEnSegundos != nil {
		t := ahora.Add(time.Duration(*expiraActualizacionEnSegundos) * time.Second)
		expRefresh = &t
	}

	item := models.Session{
		IDSesion:                     idSesion,
		ClienteID:                    clienteID,
		IDUsuario:                    idUsuario,
		NombreUsuario:                nombreUsuario,
		CorreoElectronico:            correoElectronico,
		TokenAcceso:                  &tokenAcceso,
		TokenActualizacion:           tokenActualizacion,
		IDToken:                      idToken,
		FechaExpiracionToken:         expToken.UTC().Format(time.RFC3339),
		TipoToken:                    tipoToken,
		FechaExpiracion:              expSesion.UTC().Format(time.RFC3339),
		FechaExpiracionActualizacion: s.toISO(expRefresh),
		FechaCreacion:                ahora.UTC().Format(time.RFC3339),
		FechaUltimaActividad:         ahora.UTC().Format(time.RFC3339),
		FechaRevocacion:              nil,
		Activo:                       true,
		DireccionIP:                  direccionIP,
		AgenteUsuario:                agenteUsuario,
		TTL:                          expSesion.Unix(),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return "", fmt.Errorf("error serializando sesión: %w", err)
	}

	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(id_sesion)"),
	})
	if err != nil {
		return "", fmt.Errorf("error guardando sesión en DynamoDB: %w", err)
	}

	return idSesion, nil
}

func (s *SessionService) ObtenerSesionPorID(idSesion string) (*models.Session, error) {
	out, err := s.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id_sesion": &types.AttributeValueMemberS{Value: idSesion},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("error consultando sesión: %w", err)
	}

	if out.Item == nil || len(out.Item) == 0 {
		return nil, nil
	}

	var sesion models.Session
	if err := attributevalue.UnmarshalMap(out.Item, &sesion); err != nil {
		return nil, fmt.Errorf("error deserializando sesión: %w", err)
	}

	return &sesion, nil
}

func (s *SessionService) ObtenerSesionActiva(idSesion string) (*models.Session, error) {
	sesion, err := s.ObtenerSesionPorID(idSesion)
	if err != nil {
		return nil, err
	}
	if sesion == nil {
		return nil, nil
	}

	if !sesion.Activo {
		return nil, nil
	}

	if sesion.FechaRevocacion != nil && *sesion.FechaRevocacion != "" {
		return nil, nil
	}

	if sesion.FechaExpiracion == "" {
		return nil, nil
	}

	fechaExp, err := time.Parse(time.RFC3339, sesion.FechaExpiracion)
	if err != nil {
		return nil, fmt.Errorf("fecha_expiracion inválida: %w", err)
	}

	if !fechaExp.After(s.now()) {
		return nil, nil
	}

	return sesion, nil
}

func (s *SessionService) ActualizarUltimaActividad(idSesion string) error {
	ahora := s.now().Format(time.RFC3339)

	_, err := s.client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id_sesion": &types.AttributeValueMemberS{Value: idSesion},
		},
		UpdateExpression: aws.String("SET fecha_ultima_actividad = :f"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":f": &types.AttributeValueMemberS{Value: ahora},
		},
	})
	if err != nil {
		return fmt.Errorf("error actualizando última actividad: %w", err)
	}

	return nil
}

func (s *SessionService) RevocarSesion(idSesion string) error {
	ahora := s.now().Format(time.RFC3339)

	_, err := s.client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id_sesion": &types.AttributeValueMemberS{Value: idSesion},
		},
		UpdateExpression: aws.String(`
			SET activo = :activo,
			    fecha_revocacion = :revocacion,
			    token_acceso = :nullv,
			    token_actualizacion = :nullv,
			    id_token = :nullv
		`),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":activo":     &types.AttributeValueMemberBOOL{Value: false},
			":revocacion": &types.AttributeValueMemberS{Value: ahora},
			":nullv":      &types.AttributeValueMemberNULL{Value: true},
		},
	})
	if err != nil {
		return fmt.Errorf("error revocando sesión: %w", err)
	}

	return nil
}

func (s *SessionService) CerrarSesionLocal(idSesion string) error {
	return s.RevocarSesion(idSesion)
}

func (s *SessionService) EliminarSesion(idSesion string) error {
	_, err := s.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id_sesion": &types.AttributeValueMemberS{Value: idSesion},
		},
	})
	if err != nil {
		return fmt.Errorf("error eliminando sesión: %w", err)
	}
	return nil
}

func (s *SessionService) ActualizarTokens(
	idSesion string,
	tokenAcceso string,
	tokenActualizacion *string,
	idToken *string,
	expiraEnSegundos *int,
	expiraActualizacionEnSegundos *int,
) error {
	ahora := s.now()

	expToken := ahora.Add(3600 * time.Second)
	if expiraEnSegundos != nil {
		expToken = ahora.Add(time.Duration(*expiraEnSegundos) * time.Second)
	}

	var expRefresh *time.Time
	if expiraActualizacionEnSegundos != nil {
		t := ahora.Add(time.Duration(*expiraActualizacionEnSegundos) * time.Second)
		expRefresh = &t
	}

	exprValues := map[string]types.AttributeValue{
		":ta":  &types.AttributeValueMemberS{Value: tokenAcceso},
		":fet": &types.AttributeValueMemberS{Value: expToken.UTC().Format(time.RFC3339)},
		":fua": &types.AttributeValueMemberS{Value: ahora.UTC().Format(time.RFC3339)},
	}

	updateExpr := `
		SET token_acceso = :ta,
		    fecha_expiracion_token = :fet,
		    fecha_ultima_actividad = :fua
	`

	if tokenActualizacion != nil {
		exprValues[":tr"] = &types.AttributeValueMemberS{Value: *tokenActualizacion}
		updateExpr += ", token_actualizacion = :tr"
	} else {
		exprValues[":trnull"] = &types.AttributeValueMemberNULL{Value: true}
		updateExpr += ", token_actualizacion = :trnull"
	}

	if idToken != nil {
		exprValues[":it"] = &types.AttributeValueMemberS{Value: *idToken}
		updateExpr += ", id_token = :it"
	} else {
		exprValues[":itnull"] = &types.AttributeValueMemberNULL{Value: true}
		updateExpr += ", id_token = :itnull"
	}

	if expRefresh != nil {
		exprValues[":fer"] = &types.AttributeValueMemberS{Value: expRefresh.UTC().Format(time.RFC3339)}
		updateExpr += ", fecha_expiracion_actualizacion = :fer"
	} else {
		exprValues[":fernull"] = &types.AttributeValueMemberNULL{Value: true}
		updateExpr += ", fecha_expiracion_actualizacion = :fernull"
	}

	_, err := s.client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id_sesion": &types.AttributeValueMemberS{Value: idSesion},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeValues: exprValues,
	})
	if err != nil {
		return fmt.Errorf("error actualizando tokens: %w", err)
	}

	return nil
}

func (s *SessionService) TokenAccesoVencido(sesion *models.Session) bool {
	if sesion == nil || sesion.FechaExpiracionToken == "" {
		return true
	}

	fechaExp, err := time.Parse(time.RFC3339, sesion.FechaExpiracionToken)
	if err != nil {
		return true
	}

	return !fechaExp.After(s.now())
}

func GetSesionActual(idSesion string) (*models.Session, error) {
	if idSesion == "" {
		return nil, errors.New("sesión no encontrada")
	}

	sessionService, err := NewSessionService()
	if err != nil {
		return nil, err
	}

	sesion, err := sessionService.ObtenerSesionActiva(idSesion)
	if err != nil {
		return nil, err
	}
	if sesion == nil {
		return nil, errors.New("sesión inválida o expirada")
	}

	cfg, err := GetClientConfig(sesion.ClienteID)
	if err != nil {
		return nil, err
	}

	keycloak := NewKeycloakServiceForClient(cfg)

	if sessionService.TokenAccesoVencido(sesion) {
		if sesion.TokenActualizacion == nil || *sesion.TokenActualizacion == "" {
			return nil, errors.New("refresh token no disponible")
		}

		tokenData, err := keycloak.RefreshTokens(*sesion.TokenActualizacion)
		if err != nil {
			_ = sessionService.RevocarSesion(idSesion)
			return nil, errors.New("no fue posible refrescar la sesión")
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

		err = sessionService.ActualizarTokens(
			idSesion,
			tokenData.AccessToken,
			refreshToken,
			idToken,
			&expiresIn,
			&refreshExpiresIn,
		)
		if err != nil {
			return nil, err
		}

		sesion, err = sessionService.ObtenerSesionActiva(idSesion)
		if err != nil {
			return nil, err
		}
		if sesion == nil {
			return nil, errors.New("sesión inválida o expirada")
		}
	}

	_ = sessionService.ActualizarUltimaActividad(idSesion)

	return sesion, nil
}

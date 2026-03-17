from fastapi import Request, HTTPException, status

from conf.conf import get_settings
from services.session_service import SessionService
from services.keycloak_service import KeycloakService

settings = get_settings()
session_service = SessionService()
keycloak_service = KeycloakService()


def get_id_sesion(request: Request) -> str:
    id_sesion = request.cookies.get(settings.SESSION_COOKIE_NAME)
    if not id_sesion:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Sesión no encontrada",
        )
    return id_sesion


async def get_sesion_actual(request: Request) -> dict:
    id_sesion = get_id_sesion(request)
    sesion = session_service.obtener_sesion_activa(id_sesion)

    if not sesion:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Sesión inválida o expirada",
        )

    # Si el access token ya venció, intenta refresh
    if session_service.token_acceso_vencido(sesion):
        refresh_token = sesion.get("token_actualizacion")
        if not refresh_token:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Refresh token no disponible",
            )

        try:
            token_data = await keycloak_service.refresh_tokens(refresh_token)

            session_service.actualizar_tokens(
                id_sesion=id_sesion,
                token_acceso=token_data["access_token"],
                token_actualizacion=token_data.get("refresh_token", refresh_token),
                id_token=token_data.get("id_token"),
                expira_en_segundos=token_data.get("expires_in"),
                expira_actualizacion_en_segundos=token_data.get("refresh_expires_in"),
            )

            sesion = session_service.obtener_sesion_activa(id_sesion)
            if not sesion:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="No fue posible reconstruir la sesión",
                )
        except Exception:
            session_service.revocar_sesion(id_sesion)
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="No fue posible refrescar la sesión",
            )

    session_service.actualizar_ultima_actividad(id_sesion)
    return sesion

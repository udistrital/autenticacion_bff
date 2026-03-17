from fastapi import APIRouter, Query, Request, Response, HTTPException
from fastapi.responses import RedirectResponse

from conf.conf import get_settings
from schemas.auth import LoginUrlResponse
from services.keycloak_service import KeycloakService
from services.session_service import SessionService

router = APIRouter()

settings = get_settings()
keycloak_service = KeycloakService()
session_service = SessionService()

@router.get("/login")
async def login():
    auth_url = keycloak_service.build_login_url()
    return RedirectResponse(url=auth_url)

@router.get("/login-url", response_model=LoginUrlResponse)
async def login_url() -> LoginUrlResponse:
    return LoginUrlResponse(auth_url=keycloak_service.build_login_url())


@router.get("/callback")
async def callback(
    request: Request,
    code: str = Query(...),
    state: str | None = Query(default=None),
) -> Response:
    try:
        token_data = await keycloak_service.exchange_code(code)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Error intercambiando code: {str(e)}")

    access_token = token_data["access_token"]

    userinfo = await keycloak_service.userinfo(access_token)

    id_usuario = userinfo.get("sub")
    nombre_usuario = (
        userinfo.get("preferred_username")
        or userinfo.get("name")
    )
    correo_electronico = userinfo.get("email")

    id_sesion = session_service.crear_sesion(
        id_usuario=id_usuario,
        nombre_usuario=nombre_usuario,
        correo_electronico=correo_electronico,
        token_acceso=access_token,
        token_actualizacion=token_data.get("refresh_token"),
        id_token=token_data.get("id_token"),
        tipo_token=token_data.get("token_type", "Bearer"),
        expira_en_segundos=token_data.get("expires_in"),
        expira_actualizacion_en_segundos=token_data.get("refresh_expires_in"),
        direccion_ip=request.client.host if request.client else None,
        agente_usuario=request.headers.get("user-agent"),
    )

    response = RedirectResponse(url="http://localhost:4200/#/pages")
    response.set_cookie(
        key=settings.SESSION_COOKIE_NAME,
        value=id_sesion,
        httponly=True,
        secure=settings.SESSION_COOKIE_SECURE,
        samesite=settings.SESSION_COOKIE_SAMESITE,
        path="/",
        max_age=4 * 60 * 60,
    )
    return response


@router.get("/logout")
async def logout(request: Request) -> Response:
    id_sesion = request.cookies.get(settings.SESSION_COOKIE_NAME)

    if not id_sesion:
        response = RedirectResponse(url="http://localhost:4200/#/login")
        response.delete_cookie(settings.SESSION_COOKIE_NAME, path="/")
        return response

    sesion = session_service.obtener_sesion_por_id(id_sesion)

    session_service.revocar_sesion(id_sesion)

    if sesion and sesion.get("id_token"):
        logout_url = keycloak_service.build_logout_url(
            id_token_hint=sesion["id_token"],
            post_logout_redirect_uri="http://localhost:4200/#/login",
        )
        response = RedirectResponse(url=logout_url)
    else:
        response = RedirectResponse(url="http://localhost:4200/#/login")

    response.delete_cookie(settings.SESSION_COOKIE_NAME, path="/")
    return response

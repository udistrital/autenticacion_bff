from fastapi import APIRouter, Depends, Request

from conf.security import get_sesion_actual
from schemas.user import UserMeResponse

router = APIRouter()


@router.get("", response_model=UserMeResponse)
async def get_me(sesion: dict = Depends(get_sesion_actual)) -> UserMeResponse:
    return UserMeResponse(
        id_usuario=sesion["id_usuario"],
        correo_electronico=sesion.get("correo_electronico"),
        nombre_usuario=sesion.get("nombre_usuario"),
    )

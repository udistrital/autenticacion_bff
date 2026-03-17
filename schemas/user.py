from pydantic import BaseModel, EmailStr


class UserMeResponse(BaseModel):
    id_usuario: str
    correo_electronico: EmailStr | None = None
    nombre_usuario: str | None = None

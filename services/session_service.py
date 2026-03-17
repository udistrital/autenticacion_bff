import secrets
from datetime import datetime, timedelta, timezone
from typing import Any

from conf.conf import get_settings
from conf.dynamodb import get_sesiones_table

settings = get_settings()


class SessionService:
    def __init__(self) -> None:
        self.table = get_sesiones_table()

    def _now(self) -> datetime:
        return datetime.now(timezone.utc)

    def _to_iso(self, dt: datetime | None) -> str | None:
        if dt is None:
            return None
        return dt.astimezone(timezone.utc).isoformat()

    def crear_sesion(
        self,
        id_usuario: str,
        nombre_usuario: str | None,
        correo_electronico: str | None,
        token_acceso: str,
        token_actualizacion: str | None,
        id_token: str | None,
        tipo_token: str,
        expira_en_segundos: int | None,
        expira_actualizacion_en_segundos: int | None,
        direccion_ip: str | None,
        agente_usuario: str | None,
    ) -> str:
        id_sesion = secrets.token_urlsafe(48)
        ahora = self._now()

        fecha_expiracion_token = ahora + timedelta(seconds=expira_en_segundos or 3600)
        fecha_expiracion = ahora + timedelta(hours=4)
        fecha_expiracion_actualizacion = None

        if expira_actualizacion_en_segundos:
            fecha_expiracion_actualizacion = ahora + timedelta(
                seconds=expira_actualizacion_en_segundos
            )

        item = {
            "id_sesion": id_sesion,
            "id_usuario": id_usuario,
            "nombre_usuario": nombre_usuario,
            "correo_electronico": correo_electronico,
            "token_acceso": token_acceso,
            "token_actualizacion": token_actualizacion,
            "id_token": id_token,
            "fecha_expiracion_token": self._to_iso(fecha_expiracion_token),
            "tipo_token": tipo_token,
            "fecha_expiracion": self._to_iso(fecha_expiracion),
            "fecha_expiracion_actualizacion": self._to_iso(fecha_expiracion_actualizacion),
            "fecha_creacion": self._to_iso(ahora),
            "fecha_ultima_actividad": self._to_iso(ahora),
            "fecha_revocacion": None,
            "activo": True,
            "direccion_ip": direccion_ip,
            "agente_usuario": agente_usuario,
            "ttl_epoch": int(fecha_expiracion.timestamp()),
        }

        self.table.put_item(
            Item=item,
            ConditionExpression="attribute_not_exists(id_sesion)",
        )

        return id_sesion

    def obtener_sesion_por_id(self, id_sesion: str) -> dict[str, Any] | None:
        response = self.table.get_item(
            Key={"id_sesion": id_sesion},
            ConsistentRead=True,
        )
        return response.get("Item")

    def obtener_sesion_activa(self, id_sesion: str) -> dict[str, Any] | None:
        sesion = self.obtener_sesion_por_id(id_sesion)
        if not sesion:
            return None

        if not sesion.get("activo"):
            return None

        if sesion.get("fecha_revocacion"):
            return None

        fecha_expiracion = sesion.get("fecha_expiracion")
        if not fecha_expiracion:
            return None

        if datetime.fromisoformat(fecha_expiracion) <= self._now():
            return None

        return sesion

    def actualizar_ultima_actividad(self, id_sesion: str) -> None:
        self.table.update_item(
            Key={"id_sesion": id_sesion},
            UpdateExpression="SET fecha_ultima_actividad = :f",
            ExpressionAttributeValues={
                ":f": self._to_iso(self._now()),
            },
        )

    def revocar_sesion(self, id_sesion: str) -> None:
        self.table.update_item(
            Key={"id_sesion": id_sesion},
            UpdateExpression="""
                SET activo = :activo,
                    fecha_revocacion = :revocacion,
                    token_acceso = :nullv,
                    token_actualizacion = :nullv,
                    id_token = :nullv
            """,
            ExpressionAttributeValues={
                ":activo": False,
                ":revocacion": self._to_iso(self._now()),
                ":nullv": None,
            },
        )

    def cerrar_sesion_local(self, id_sesion: str) -> None:
        self.revocar_sesion(id_sesion)

    def eliminar_sesion(self, id_sesion: str) -> None:
        self.table.delete_item(Key={"id_sesion": id_sesion})
    
    def actualizar_tokens(
        self,
        id_sesion: str,
        token_acceso: str,
        token_actualizacion: str | None,
        id_token: str | None,
        expira_en_segundos: int | None,
        expira_actualizacion_en_segundos: int | None,
    ) -> None:
        ahora = self._now()
        fecha_expiracion_token = ahora + timedelta(seconds=expira_en_segundos or 3600)

        fecha_expiracion_actualizacion = None
        if expira_actualizacion_en_segundos:
            fecha_expiracion_actualizacion = ahora + timedelta(
                seconds=expira_actualizacion_en_segundos
            )

        self.table.update_item(
            Key={"id_sesion": id_sesion},
            UpdateExpression="""
                SET token_acceso = :ta,
                    token_actualizacion = :tr,
                    id_token = :it,
                    fecha_expiracion_token = :fet,
                    fecha_expiracion_actualizacion = :fer,
                    fecha_ultima_actividad = :fua
            """,
            ExpressionAttributeValues={
                ":ta": token_acceso,
                ":tr": token_actualizacion,
                ":it": id_token,
                ":fet": self._to_iso(fecha_expiracion_token),
                ":fer": self._to_iso(fecha_expiracion_actualizacion),
                ":fua": self._to_iso(ahora),
            },
        )

    def token_acceso_vencido(self, sesion: dict[str, Any]) -> bool:
        fecha_expiracion_token = sesion.get("fecha_expiracion_token")
        if not fecha_expiracion_token:
            return True
        return datetime.fromisoformat(fecha_expiracion_token) <= self._now()

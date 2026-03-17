from functools import lru_cache
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=None,
        case_sensitive=True,
        extra="ignore",
    )

    APP_NAME: str = "autenticacion-bff"
    APP_ENV: str = "dev"
    APP_PORT: int = 8000
    DEBUG: bool = True
    API_V1_PREFIX: str = "/api/v1"

    KEYCLOAK_BASE_URL: str
    KEYCLOAK_REALM: str
    KEYCLOAK_CLIENT_ID: str
    KEYCLOAK_CLIENT_SECRET: str
    KEYCLOAK_REDIRECT_URI: str

    SESSION_COOKIE_NAME: str = "id_sesion"
    SESSION_COOKIE_SECURE: bool = False
    SESSION_COOKIE_SAMESITE: str = "lax"

    BACKEND_CORS_ORIGINS: list[str] = [
        "http://localhost:4200",
        "http://localhost:3000",
    ]

    AWS_REGION: str = "us-east-1"
    DYNAMODB_TABLE_SESIONES: str = "sesiones"
    AWS_ACCESS_KEY_ID: str | None = None
    AWS_SECRET_ACCESS_KEY: str | None = None
    AWS_SESSION_TOKEN: str | None = None
    DYNAMODB_ENDPOINT_URL: str | None = None

@lru_cache
def get_settings() -> Settings:
    return Settings()
from urllib.parse import urlencode

import httpx

from conf.conf import get_settings

settings = get_settings()


class KeycloakService:
    def __init__(self) -> None:
        self.base = settings.KEYCLOAK_BASE_URL.rstrip("/")
        self.realm = settings.KEYCLOAK_REALM
        self.client_id = settings.KEYCLOAK_CLIENT_ID
        self.client_secret = settings.KEYCLOAK_CLIENT_SECRET
        self.redirect_uri = settings.KEYCLOAK_REDIRECT_URI

    @property
    def realm_url(self) -> str:
        return f"{self.base}/realms/{self.realm}"

    def build_login_url(self, state: str = "estado-inicial") -> str:
        params = urlencode({
            "client_id": self.client_id,
            "response_type": "code",
            "scope": "openid profile email",
            "redirect_uri": self.redirect_uri,
            "state": state,
        })
        return f"{self.realm_url}/protocol/openid-connect/auth?{params}"

    def build_logout_url(self, id_token_hint: str, post_logout_redirect_uri: str) -> str:
        params = urlencode({
            "id_token_hint": id_token_hint,
            "post_logout_redirect_uri": post_logout_redirect_uri,
        })
        return f"{self.realm_url}/protocol/openid-connect/logout?{params}"

    async def exchange_code(self, code: str) -> dict:
        token_url = f"{self.realm_url}/protocol/openid-connect/token"
        data = {
            "grant_type": "authorization_code",
            "client_id": self.client_id,
            "client_secret": self.client_secret,
            "code": code,
            "redirect_uri": self.redirect_uri,
        }

        async with httpx.AsyncClient(timeout=15.0) as client:
            response = await client.post(token_url, data=data)

            print("=== TOKEN URL ===")
            print(token_url)
            print("=== TOKEN DATA ===")
            print(data)
            print("=== TOKEN STATUS ===")
            print(response.status_code)
            print("=== TOKEN BODY ===")
            print(response.text)

            response.raise_for_status()
            return response.json()

    async def userinfo(self, access_token: str) -> dict:
        userinfo_url = f"{self.realm_url}/protocol/openid-connect/userinfo"
        headers = {"Authorization": f"Bearer {access_token}"}

        async with httpx.AsyncClient(timeout=15.0) as client:
            response = await client.get(userinfo_url, headers=headers)
            response.raise_for_status()
            return response.json()

    async def refresh_tokens(self, refresh_token: str) -> dict:
        token_url = f"{self.realm_url}/protocol/openid-connect/token"
        data = {
            "grant_type": "refresh_token",
            "client_id": self.client_id,
            "client_secret": self.client_secret,
            "refresh_token": refresh_token,
        }

        async with httpx.AsyncClient(timeout=15.0) as client:
            response = await client.post(token_url, data=data)
            response.raise_for_status()
            return response.json()

from pydantic import BaseModel


class LoginUrlResponse(BaseModel):
    auth_url: str

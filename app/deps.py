from dataclasses import dataclass
from typing import Iterator

import jwt
from fastapi import Depends, HTTPException
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from sqlalchemy.orm import Session

from .core import blocklist
from .core.database import SessionLocal
from .core.security import decode_token

_bearer = HTTPBearer(auto_error=False)


def get_db() -> Iterator[Session]:
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@dataclass
class TokenData:
    identity: str
    jti: str
    type: str

    @property
    def user_id(self) -> int:
        return int(self.identity)


def _authenticate(creds: HTTPAuthorizationCredentials | None, expected_type: str) -> TokenData:
    if creds is None:
        raise HTTPException(401, "Missing Authorization Header")
    try:
        payload = decode_token(creds.credentials)
    except jwt.ExpiredSignatureError:
        raise HTTPException(401, "Token has expired")
    except jwt.PyJWTError:
        raise HTTPException(422, "Invalid token")
    if blocklist.is_revoked(payload["jti"]):
        raise HTTPException(401, "Token has been revoked")
    if payload.get("type") != expected_type:
        raise HTTPException(422, "Only %s tokens are allowed" % expected_type)
    return TokenData(identity=str(payload["sub"]), jti=payload["jti"], type=payload["type"])


def get_current_token(creds: HTTPAuthorizationCredentials | None = Depends(_bearer)) -> TokenData:
    return _authenticate(creds, "access")


def get_refresh_token(creds: HTTPAuthorizationCredentials | None = Depends(_bearer)) -> TokenData:
    return _authenticate(creds, "refresh")


def unauthorized() -> HTTPException:
    return HTTPException(401, "Unauthorized")

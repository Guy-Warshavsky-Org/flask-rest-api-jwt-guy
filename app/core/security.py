import uuid
from datetime import datetime, timedelta, timezone

import jwt

from .config import settings

ALGORITHM = "HS256"


def _create_token(identity: str, token_type: str, expires: timedelta) -> str:
    now = datetime.now(timezone.utc)
    payload = {
        "sub": identity,
        "jti": str(uuid.uuid4()),
        "type": token_type,
        "iat": now,
        "nbf": now,
        "exp": now + expires,
    }
    return jwt.encode(payload, settings.secret_key, algorithm=ALGORITHM)


def create_access_token(identity: str) -> str:
    return _create_token(identity, "access", timedelta(minutes=settings.access_token_expires_minutes))


def create_refresh_token(identity: str) -> str:
    return _create_token(identity, "refresh", timedelta(days=settings.refresh_token_expires_days))


def decode_token(token: str) -> dict:
    return jwt.decode(token, settings.secret_key, algorithms=[ALGORITHM], options={"require": ["sub", "jti", "exp"]})

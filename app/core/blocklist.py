"""In-memory token blocklist (revoked jtis)."""

_revoked: set[str] = set()


def revoke(jti: str) -> None:
    _revoked.add(jti)


def is_revoked(jti: str) -> bool:
    return jti in _revoked


def reset() -> None:
    _revoked.clear()

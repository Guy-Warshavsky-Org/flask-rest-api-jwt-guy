import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from app import create_app, db, jwt_token_blacklist  # noqa: E402
from config import DevelopmentConfig  # noqa: E402


@pytest.fixture()
def app(tmp_path, monkeypatch):
    """App built by the factory against a temporary SQLite database."""
    db_path = tmp_path / "test.sqlite"
    monkeypatch.setattr(
        DevelopmentConfig, "SQLALCHEMY_DATABASE_URI", f"sqlite:///{db_path}"
    )
    jwt_token_blacklist.clear()
    application = create_app("development")
    application.config["TESTING"] = True
    yield application
    with application.app_context():
        db.session.remove()
        db.engine.dispose()
    jwt_token_blacklist.clear()


@pytest.fixture()
def client(app):
    return app.test_client()


class Auth:
    def __init__(self, client):
        self.client = client

    def register(self, username, password="pw12345"):
        return self.client.post(
            "/user/register", json={"username": username, "password": password}
        )

    def login(self, username, password="pw12345"):
        return self.client.post(
            "/user/login", json={"username": username, "password": password}
        )

    def create_user(self, username, password="pw12345"):
        """Register + login; returns (user_id, access, refresh)."""
        uid = self.register(username, password).get_json()["id"]
        tokens = self.login(username, password).get_json()
        return uid, tokens["access_token"], tokens["refresh_token"]


@pytest.fixture()
def auth(client):
    return Auth(client)


def bearer(token):
    return {"Authorization": f"Bearer {token}"}


@pytest.fixture()
def user_a(auth):
    uid, access, refresh = auth.create_user("alice")
    return {"id": uid, "access": access, "refresh": refresh, "h": bearer(access)}


@pytest.fixture()
def user_b(auth):
    uid, access, refresh = auth.create_user("bob")
    return {"id": uid, "access": access, "refresh": refresh, "h": bearer(access)}

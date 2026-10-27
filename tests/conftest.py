import pytest
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool

from app.core import blocklist
from app.core.database import Base, create_db
from app.deps import get_db
from app.main import app


@pytest.fixture()
def client():
    engine = create_engine(
        "sqlite://", connect_args={"check_same_thread": False}, poolclass=StaticPool
    )
    create_db(bind=engine)
    Session = sessionmaker(bind=engine, autoflush=False, expire_on_commit=False)

    def override():
        db = Session()
        try:
            yield db
        finally:
            db.close()

    app.dependency_overrides[get_db] = override
    blocklist.reset()
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()
    Base.metadata.drop_all(engine)


def register_login(client, username="alice", password="pw"):
    r = client.post("/user/register", json={"username": username, "password": password})
    assert r.status_code == 201
    uid = r.json()["id"]
    t = client.post("/user/login", json={"username": username, "password": password}).json()
    return uid, t


def auth(token):
    return {"Authorization": f"Bearer {token}"}


@pytest.fixture()
def alice(client):
    uid, t = register_login(client, "alice")
    return uid, t, auth(t["access_token"])


@pytest.fixture()
def bob(client):
    uid, t = register_login(client, "bob")
    return uid, t, auth(t["access_token"])

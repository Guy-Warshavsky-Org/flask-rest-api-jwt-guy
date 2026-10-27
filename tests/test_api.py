import time
import jwt as pyjwt

from app.core.config import settings
from app.core.security import create_access_token
from .conftest import auth


# ---- health
def test_health(client):
    r = client.get("/health/")
    assert r.status_code == 200
    assert r.json() == {"status": "healthy", "database": "healthy"}


# ---- users / auth
def test_register_and_duplicate(client):
    r = client.post("/user/register", json={"username": "a", "password": "p"})
    assert r.status_code == 201
    assert r.json() == {"id": 1, "username": "a"}
    r = client.post("/user/register", json={"username": "a", "password": "p"})
    assert r.status_code == 400 and r.json() == {"message": "User exists"}


def test_register_missing_field_422(client):
    assert client.post("/user/register", json={"username": "a"}).status_code == 422


def test_login(client):
    client.post("/user/register", json={"username": "a", "password": "p"})
    r = client.post("/user/login", json={"username": "a", "password": "p"})
    assert r.status_code == 200
    body = r.json()
    acc = pyjwt.decode(body["access_token"], settings.secret_key, algorithms=["HS256"])
    ref = pyjwt.decode(body["refresh_token"], settings.secret_key, algorithms=["HS256"])
    assert acc["sub"] == "1" and acc["type"] == "access" and acc["jti"]
    assert ref["type"] == "refresh"
    assert acc["exp"] - acc["iat"] == 15 * 60
    assert ref["exp"] - ref["iat"] == 30 * 86400
    for bad in ({"username": "a", "password": "x"}, {"username": "zz", "password": "p"}):
        r = client.post("/user/login", json=bad)
        assert r.status_code == 401 and r.json() == {"message": "Invalid credentials"}


def test_get_delete_user_and_isolation(client, alice, bob):
    aid, _, ah = alice
    bid, _, bh = bob
    r = client.get(f"/user/{aid}", headers=ah)
    assert r.status_code == 200 and r.json() == {"id": aid, "username": "alice"}
    assert "password_hash" not in r.json()
    r = client.get(f"/user/{bid}", headers=ah)
    assert r.status_code == 401 and r.json() == {"message": "Unauthorized"}
    assert client.delete(f"/user/{bid}", headers=ah).status_code == 401
    assert client.get("/user/999", headers=ah).status_code == 404
    r = client.delete(f"/user/{aid}", headers=ah)
    assert r.status_code == 200 and r.json() == {"message": "Deleted"}
    assert client.get(f"/user/{aid}", headers=bh).status_code == 404


def test_logout_revokes_token(client, alice):
    _, _, ah = alice
    r = client.post("/user/logout", headers=ah)
    assert r.status_code == 200 and r.json() == {"message": "Logged out"}
    r = client.get("/store/s", headers=ah)
    assert r.status_code == 401 and "message" in r.json()


def test_refresh(client, alice):
    _, t, ah = alice
    r = client.post("/user/refresh", headers=auth(t["refresh_token"]))
    assert r.status_code == 200
    assert list(r.json()) == ["access_token"]
    assert client.get("/store/s", headers=auth(r.json()["access_token"])).status_code == 200
    # access token cannot refresh; refresh token cannot access
    assert client.post("/user/refresh", headers=ah).status_code == 422
    assert client.get("/store/s", headers=auth(t["refresh_token"])).status_code == 422


def test_auth_failure_modes(client):
    r = client.get("/store/s")
    assert r.status_code == 401 and "message" in r.json()
    r = client.get("/store/s", headers=auth("garbage"))
    assert r.status_code == 422 and "message" in r.json()
    bad = pyjwt.encode({"sub": "1", "jti": "x", "type": "access", "exp": time.time() + 60}, "wrong", algorithm="HS256")
    assert client.get("/store/s", headers=auth(bad)).status_code == 422
    expired = pyjwt.encode(
        {"sub": "1", "jti": "y", "type": "access", "exp": int(time.time()) - 10}, settings.secret_key, algorithm="HS256"
    )
    r = client.get("/store/s", headers=auth(expired))
    assert r.status_code == 401 and "message" in r.json()


# ---- stores
def test_store_crud(client, alice, bob):
    _, _, ah = alice
    _, _, bh = bob
    r = client.post("/store/", json={"name": "S"}, headers=ah)
    assert r.status_code == 201 and r.json() == {"id": 1, "name": "S", "user_id": 1}
    assert client.get("/store/1", headers=ah).json() == {"id": 1, "name": "S", "user_id": 1}
    assert client.get("/store/s", headers=ah).json() == [{"id": 1, "name": "S", "user_id": 1}]
    assert client.get("/store/s", headers=bh).json() == []
    r = client.get("/store/1", headers=bh)
    assert r.status_code == 401 and r.json() == {"message": "Unauthorized"}
    assert client.delete("/store/1", headers=bh).status_code == 401
    assert client.get("/store/99", headers=ah).status_code == 404
    assert client.post("/store/", json={}, headers=ah).status_code == 422
    r = client.delete("/store/1", headers=ah)
    assert r.json() == {"message": "Deleted"}
    assert client.get("/store/1", headers=ah).status_code == 404


# ---- tags
def test_tag_crud(client, alice, bob):
    _, _, ah = alice
    _, _, bh = bob
    client.post("/store/", json={"name": "S"}, headers=ah)
    r = client.post("/tag/store/1", json={"name": "t"}, headers=ah)
    assert r.status_code == 201 and r.json() == {"id": 1, "name": "t", "store_id": 1}
    assert client.get("/tag/1", headers=ah).json() == {"id": 1, "name": "t", "store_id": 1}
    assert client.get("/tag/store/1/s", headers=ah).json() == [{"id": 1, "name": "t", "store_id": 1}]
    assert client.get("/tag/1", headers=bh).status_code == 401
    assert client.get("/tag/store/1/s", headers=bh).status_code == 401
    assert client.post("/tag/store/1", json={"name": "x"}, headers=bh).status_code == 401
    assert client.delete("/tag/1", headers=bh).status_code == 401
    assert client.post("/tag/store/99", json={"name": "x"}, headers=ah).status_code == 404
    assert client.get("/tag/99", headers=ah).status_code == 404
    assert client.get("/tag/store/99/s", headers=ah).status_code == 404
    assert client.delete("/tag/1", headers=ah).json() == {"message": "Deleted"}
    assert client.get("/tag/1", headers=ah).status_code == 404


# ---- items
def test_item_crud(client, alice, bob):
    _, _, ah = alice
    _, _, bh = bob
    client.post("/store/", json={"name": "S1"}, headers=ah)
    client.post("/store/", json={"name": "S2"}, headers=ah)
    client.post("/store/", json={"name": "B"}, headers=bh)
    r = client.post("/item/", json={"name": "i", "price": 2.5, "store_id": 1}, headers=ah)
    assert r.status_code == 201 and r.json() == {"id": 1, "name": "i", "price": 2.5, "store_id": 1}
    assert client.get("/item/s", headers=ah).json() == [r.json()]
    assert client.get("/item/s", headers=bh).json() == []
    assert client.get("/item/1", headers=ah).json() == r.json()
    assert client.get("/item/1", headers=bh).status_code == 401
    assert client.post("/item/", json={"name": "x", "price": 1, "store_id": 3}, headers=ah).status_code == 401
    assert client.post("/item/", json={"name": "x", "price": 1, "store_id": 99}, headers=ah).status_code == 404
    assert client.post("/item/", json={"name": "x"}, headers=ah).status_code == 422
    # update
    r = client.put("/item/1", json={"price": 9}, headers=ah)
    assert r.json() == {"id": 1, "name": "i", "price": 9.0, "store_id": 1}
    r = client.put("/item/1", json={"name": "n", "store_id": 2}, headers=ah)
    assert r.json() == {"id": 1, "name": "n", "price": 9.0, "store_id": 2}
    assert client.put("/item/1", json={"store_id": 3}, headers=ah).status_code == 401
    assert client.put("/item/1", json={"store_id": 99}, headers=ah).status_code == 404
    assert client.put("/item/1", json={"name": "h"}, headers=bh).status_code == 401
    assert client.put("/item/99", json={}, headers=ah).status_code == 404
    assert client.delete("/item/1", headers=bh).status_code == 401
    assert client.delete("/item/1", headers=ah).json() == {"message": "Deleted"}
    assert client.get("/item/1", headers=ah).status_code == 404


def test_item_update_explicit_null_rejected(client, alice):
    _, _, ah = alice
    client.post("/store/", json={"name": "S"}, headers=ah)
    client.post("/item/", json={"name": "i", "price": 2.5, "store_id": 1}, headers=ah)
    for field in ("name", "price", "store_id"):
        assert client.put("/item/1", json={field: None}, headers=ah).status_code == 422
    assert client.get("/item/1", headers=ah).json() == {"id": 1, "name": "i", "price": 2.5, "store_id": 1}
    assert client.put("/item/1", json={}, headers=ah).status_code == 200


def test_wrong_type_revoked_token_returns_422(client, alice):
    _, t, ah = alice
    client.post("/user/logout", headers=ah)
    assert client.post("/user/refresh", headers=ah).status_code == 422


# ---- cascades
def test_cascade_delete_store_and_user(client, alice):
    uid, _, ah = alice
    client.post("/store/", json={"name": "S"}, headers=ah)
    client.post("/tag/store/1", json={"name": "t"}, headers=ah)
    client.post("/item/", json={"name": "i", "price": 1, "store_id": 1}, headers=ah)
    client.delete("/store/1", headers=ah)
    assert client.get("/tag/1", headers=ah).status_code == 404
    assert client.get("/item/1", headers=ah).status_code == 404

    client.post("/store/", json={"name": "S"}, headers=ah)
    client.post("/tag/store/2", json={"name": "t"}, headers=ah)
    client.delete(f"/user/{uid}", headers=ah)
    from app.deps import get_db
    from app.main import app
    from app.models import Item, Store, Tag
    db = next(app.dependency_overrides[get_db]())
    assert db.query(Store).count() == 0 and db.query(Tag).count() == 0 and db.query(Item).count() == 0


def test_token_for_deleted_identity_type(client):
    # token bearer with valid signature works at the auth layer
    t = create_access_token("42")
    assert client.get("/store/s", headers=auth(t)).json() == []

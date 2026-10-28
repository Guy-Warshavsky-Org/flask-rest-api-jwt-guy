from conftest import bearer


def test_register_success(auth):
    r = auth.register("carol")
    assert r.status_code == 201
    body = r.get_json()
    assert body["username"] == "carol"
    assert "password_hash" not in body


def test_register_duplicate(auth):
    auth.register("carol")
    r = auth.register("carol")
    assert r.status_code == 400
    assert r.get_json()["message"] == "User exists"


def test_login_success_and_failure(auth):
    auth.register("carol")
    r = auth.login("carol")
    assert r.status_code == 200
    assert set(r.get_json()) == {"access_token", "refresh_token"}
    assert auth.login("carol", "wrong").status_code == 401
    assert auth.login("nobody").status_code == 401


def test_refresh_returns_usable_access_token(client, user_a):
    r = client.post("/user/refresh", headers=bearer(user_a["refresh"]))
    assert r.status_code == 200
    new_access = r.get_json()["access_token"]
    r = client.get(f"/user/{user_a['id']}", headers=bearer(new_access))
    assert r.status_code == 200


def test_refresh_rejects_access_token(client, user_a):
    r = client.post("/user/refresh", headers=user_a["h"])
    assert r.status_code == 422


def test_protected_route_requires_token(client, user_a):
    assert client.get(f"/user/{user_a['id']}").status_code == 401


def test_logout_revokes_token(client, user_a):
    r = client.post("/user/logout", headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["message"] == "Logged out"
    r = client.get(f"/user/{user_a['id']}", headers=user_a["h"])
    assert r.status_code == 401


def test_get_user(client, user_a):
    r = client.get(f"/user/{user_a['id']}", headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["username"] == "alice"


def test_get_user_not_found(client, user_a):
    assert client.get("/user/9999", headers=user_a["h"]).status_code == 404


def test_get_other_user_unauthorized(client, user_a, user_b):
    r = client.get(f"/user/{user_b['id']}", headers=user_a["h"])
    assert r.status_code == 401


def test_delete_user(client, auth, user_a):
    r = client.delete(f"/user/{user_a['id']}", headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["message"] == "Deleted"
    assert auth.login("alice").status_code == 401


def test_delete_other_user_unauthorized(client, auth, user_a, user_b):
    r = client.delete(f"/user/{user_b['id']}", headers=user_a["h"])
    assert r.status_code == 401
    assert auth.login("bob").status_code == 200

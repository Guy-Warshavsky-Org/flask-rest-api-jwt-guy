from app import db
from app.models import Item, Store, Tag, User


def make_store(client, user, name="Shop"):
    r = client.post("/store/", json={"name": name}, headers=user["h"])
    assert r.status_code == 201
    return r.get_json()["id"]


def make_item(client, user, store_id, name="Widget", price=9.5):
    r = client.post(
        "/item/",
        json={"name": name, "price": price, "store_id": store_id},
        headers=user["h"],
    )
    assert r.status_code == 201
    return r.get_json()["id"]


def make_tag(client, user, store_id, name="sale"):
    r = client.post(f"/tag/store/{store_id}", json={"name": name}, headers=user["h"])
    assert r.status_code == 201
    return r.get_json()["id"]


# ---- stores ----

def test_store_crud(client, user_a):
    sid = make_store(client, user_a, "Alpha")
    r = client.get(f"/store/{sid}", headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["name"] == "Alpha"
    assert r.get_json()["user_id"] == user_a["id"]

    make_store(client, user_a, "Beta")
    r = client.get("/store/s", headers=user_a["h"])
    assert sorted(s["name"] for s in r.get_json()) == ["Alpha", "Beta"]

    assert client.delete(f"/store/{sid}", headers=user_a["h"]).status_code == 200
    assert client.get(f"/store/{sid}", headers=user_a["h"]).status_code == 404


def test_store_requires_auth(client):
    assert client.post("/store/", json={"name": "x"}).status_code == 401


# ---- items ----

def test_item_crud(client, user_a):
    sid = make_store(client, user_a)
    iid = make_item(client, user_a, sid)
    r = client.get(f"/item/{iid}", headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["name"] == "Widget"
    assert r.get_json()["price"] == 9.5

    r = client.put(
        f"/item/{iid}", json={"name": "Gadget", "price": 12.0}, headers=user_a["h"]
    )
    assert r.status_code == 200
    assert r.get_json()["name"] == "Gadget"
    assert r.get_json()["price"] == 12.0

    r = client.get("/item/s", headers=user_a["h"])
    assert [i["id"] for i in r.get_json()] == [iid]

    assert client.delete(f"/item/{iid}", headers=user_a["h"]).status_code == 200
    assert client.get(f"/item/{iid}", headers=user_a["h"]).status_code == 404


def test_item_move_to_other_store(client, user_a):
    s1 = make_store(client, user_a, "one")
    s2 = make_store(client, user_a, "two")
    iid = make_item(client, user_a, s1)
    r = client.put(f"/item/{iid}", json={"store_id": s2}, headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["store_id"] == s2


# ---- tags ----

def test_tag_crud(client, user_a):
    sid = make_store(client, user_a)
    tid = make_tag(client, user_a, sid, "new")
    r = client.get(f"/tag/{tid}", headers=user_a["h"])
    assert r.status_code == 200
    assert r.get_json()["name"] == "new"
    assert r.get_json()["store_id"] == sid

    make_tag(client, user_a, sid, "hot")
    r = client.get(f"/tag/store/{sid}/s", headers=user_a["h"])
    assert sorted(t["name"] for t in r.get_json()) == ["hot", "new"]

    assert client.delete(f"/tag/{tid}", headers=user_a["h"]).status_code == 200
    assert client.get(f"/tag/{tid}", headers=user_a["h"]).status_code == 404


# ---- per-user authorization ----

def test_cannot_access_other_users_store(client, user_a, user_b):
    sid = make_store(client, user_a)
    assert client.get(f"/store/{sid}", headers=user_b["h"]).status_code == 401
    assert client.delete(f"/store/{sid}", headers=user_b["h"]).status_code == 401
    assert client.get("/store/s", headers=user_b["h"]).get_json() == []
    assert client.get(f"/store/{sid}", headers=user_a["h"]).status_code == 200


def test_cannot_access_other_users_items(client, user_a, user_b):
    sid = make_store(client, user_a)
    iid = make_item(client, user_a, sid)
    assert client.get(f"/item/{iid}", headers=user_b["h"]).status_code == 401
    assert (
        client.put(f"/item/{iid}", json={"name": "x"}, headers=user_b["h"]).status_code
        == 401
    )
    assert client.delete(f"/item/{iid}", headers=user_b["h"]).status_code == 401
    r = client.post(
        "/item/",
        json={"name": "x", "price": 1, "store_id": sid},
        headers=user_b["h"],
    )
    assert r.status_code == 401
    assert client.get("/item/s", headers=user_b["h"]).get_json() == []


def test_cannot_move_item_into_other_users_store(client, user_a, user_b):
    mine = make_store(client, user_b, "mine")
    theirs = make_store(client, user_a, "theirs")
    iid = make_item(client, user_b, mine)
    r = client.put(f"/item/{iid}", json={"store_id": theirs}, headers=user_b["h"])
    assert r.status_code == 401


def test_cannot_access_other_users_tags(client, user_a, user_b):
    sid = make_store(client, user_a)
    tid = make_tag(client, user_a, sid)
    assert client.get(f"/tag/{tid}", headers=user_b["h"]).status_code == 401
    assert client.delete(f"/tag/{tid}", headers=user_b["h"]).status_code == 401
    assert client.get(f"/tag/store/{sid}/s", headers=user_b["h"]).status_code == 401
    r = client.post(f"/tag/store/{sid}", json={"name": "x"}, headers=user_b["h"])
    assert r.status_code == 401


# ---- cascade deletion ----

def test_delete_store_cascades_to_items_and_tags(app, client, user_a):
    sid = make_store(client, user_a)
    make_item(client, user_a, sid)
    make_tag(client, user_a, sid)
    assert client.delete(f"/store/{sid}", headers=user_a["h"]).status_code == 200
    with app.app_context():
        assert Store.query.count() == 0
        assert Item.query.count() == 0
        assert Tag.query.count() == 0


def test_delete_user_cascades_to_everything(app, client, user_a, user_b):
    sid = make_store(client, user_a)
    make_item(client, user_a, sid)
    make_tag(client, user_a, sid)
    other = make_store(client, user_b, "keep")
    make_item(client, user_b, other)

    assert client.delete(f"/user/{user_a['id']}", headers=user_a["h"]).status_code == 200
    with app.app_context():
        assert db.session.get(User, user_a["id"]) is None
        assert Store.query.count() == 1
        assert Item.query.count() == 1
        assert Tag.query.count() == 0
        assert Store.query.one().user_id == user_b["id"]


# ---- health ----

def test_health(client):
    r = client.get("/health/")
    assert r.status_code == 200
    assert r.get_json() == {"status": "healthy", "database": "healthy"}

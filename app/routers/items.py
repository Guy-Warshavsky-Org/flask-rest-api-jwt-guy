from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..deps import TokenData, get_current_token, get_db, unauthorized
from ..models import Item, Store
from ..schemas import ItemIn, ItemOut, ItemUpdate

router = APIRouter(prefix="/item")


def _get_own_store(store_id: int, token: TokenData, db: Session) -> Store:
    store = db.get(Store, store_id)
    if store is None:
        raise HTTPException(404, "Not Found")
    if store.user_id != token.user_id:
        raise unauthorized()
    return store


def _get_own_item(id: int, token: TokenData, db: Session) -> Item:
    item = db.get(Item, id)
    if item is None:
        raise HTTPException(404, "Not Found")
    if item.store.user_id != token.user_id:
        raise unauthorized()
    return item


@router.post("/", status_code=201, response_model=ItemOut)
def create_item(data: ItemIn, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    _get_own_store(data.store_id, token, db)
    item = Item(name=data.name, price=data.price, store_id=data.store_id)
    db.add(item)
    db.commit()
    return item


@router.get("/s", response_model=list[ItemOut])
def get_all_items(token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    store_ids = db.scalars(select(Store.id).filter_by(user_id=token.user_id)).all()
    return db.scalars(select(Item).where(Item.store_id.in_(store_ids))).all()


@router.get("/{id}", response_model=ItemOut)
def get_item(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    return _get_own_item(id, token, db)


@router.put("/{id}", response_model=ItemOut)
def update_item(id: int, data: ItemUpdate, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    item = _get_own_item(id, token, db)
    if data.name is not None:
        item.name = data.name
    if data.price is not None:
        item.price = data.price
    if data.store_id is not None:
        _get_own_store(data.store_id, token, db)
        item.store_id = data.store_id
    db.commit()
    return item


@router.delete("/{id}")
def delete_item(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    item = _get_own_item(id, token, db)
    db.delete(item)
    db.commit()
    return {"message": "Deleted"}

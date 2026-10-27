from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..deps import TokenData, get_current_token, get_db, unauthorized
from ..models import Store, Tag
from ..schemas import TagIn, TagOut

router = APIRouter(prefix="/tag")


def _get_own_store(store_id: int, token: TokenData, db: Session) -> Store:
    store = db.get(Store, store_id)
    if store is None:
        raise HTTPException(404, "Not Found")
    if store.user_id != token.user_id:
        raise unauthorized()
    return store


@router.post("/store/{store_id}", status_code=201, response_model=TagOut)
def create_tag(store_id: int, data: TagIn, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    _get_own_store(store_id, token, db)
    tag = Tag(name=data.name, store_id=store_id)
    db.add(tag)
    db.commit()
    return tag


@router.get("/store/{store_id}/s", response_model=list[TagOut])
def get_tags_in_store(store_id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    _get_own_store(store_id, token, db)
    return db.scalars(select(Tag).filter_by(store_id=store_id)).all()


def _get_own_tag(id: int, token: TokenData, db: Session) -> Tag:
    tag = db.get(Tag, id)
    if tag is None:
        raise HTTPException(404, "Not Found")
    if tag.store.user_id != token.user_id:
        raise unauthorized()
    return tag


@router.get("/{id}", response_model=TagOut)
def get_tag(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    return _get_own_tag(id, token, db)


@router.delete("/{id}")
def delete_tag(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    tag = _get_own_tag(id, token, db)
    db.delete(tag)
    db.commit()
    return {"message": "Deleted"}

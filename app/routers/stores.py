from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..deps import TokenData, get_current_token, get_db, get_own_store
from ..models import Store
from ..schemas import StoreIn, StoreOut

router = APIRouter(prefix="/store")


@router.post("/", status_code=201, response_model=StoreOut)
def create_store(data: StoreIn, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    store = Store(name=data.name, user_id=token.user_id)
    db.add(store)
    db.commit()
    return store


@router.get("/s", response_model=list[StoreOut])
def get_all_stores(token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    return db.scalars(select(Store).filter_by(user_id=token.user_id)).all()


@router.get("/{id}", response_model=StoreOut)
def get_store(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    return get_own_store(id, token, db)


@router.delete("/{id}")
def delete_store(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    store = get_own_store(id, token, db)
    db.delete(store)
    db.commit()
    return {"message": "Deleted"}

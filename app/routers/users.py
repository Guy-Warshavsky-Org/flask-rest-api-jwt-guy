from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..core import blocklist
from ..core.security import create_access_token, create_refresh_token
from ..deps import TokenData, get_current_token, get_db, get_refresh_token, unauthorized
from ..models import User
from ..schemas import Credentials, UserOut

router = APIRouter(prefix="/user")


@router.post("/register", status_code=201, response_model=UserOut)
def register(data: Credentials, db: Session = Depends(get_db)):
    if db.scalar(select(User).filter_by(username=data.username)):
        raise HTTPException(400, "User exists")
    user = User(username=data.username)
    user.set_password(data.password)
    db.add(user)
    db.commit()
    return user


@router.post("/login")
def login(data: Credentials, db: Session = Depends(get_db)):
    user = db.scalar(select(User).filter_by(username=data.username))
    if user and user.check_password(data.password):
        return {
            "access_token": create_access_token(str(user.id)),
            "refresh_token": create_refresh_token(str(user.id)),
        }
    raise HTTPException(401, "Invalid credentials")


@router.post("/logout")
def logout(token: TokenData = Depends(get_current_token)):
    blocklist.revoke(token.jti)
    return {"message": "Logged out"}


@router.post("/refresh")
def refresh(token: TokenData = Depends(get_refresh_token)):
    return {"access_token": create_access_token(token.identity)}


def _get_own_user(id: int, token: TokenData, db: Session) -> User:
    user = db.get(User, id)
    if user is None:
        raise HTTPException(404, "Not Found")
    if token.user_id != user.id:
        raise unauthorized()
    return user


@router.get("/{id}", response_model=UserOut)
def get_user(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    return _get_own_user(id, token, db)


@router.delete("/{id}")
def delete_user(id: int, token: TokenData = Depends(get_current_token), db: Session = Depends(get_db)):
    user = _get_own_user(id, token, db)
    db.delete(user)
    db.commit()
    return {"message": "Deleted"}

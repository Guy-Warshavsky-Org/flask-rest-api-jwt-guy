from fastapi import APIRouter, Depends
from fastapi.responses import JSONResponse
from sqlalchemy import text
from sqlalchemy.orm import Session

from ..deps import get_db

router = APIRouter(prefix="/health")


@router.get("/")
def health_check(db: Session = Depends(get_db)):
    """Returns 200 when the database is reachable, otherwise 503."""
    try:
        db.execute(text("SELECT 1"))
        ok = True
    except Exception:
        ok = False
    status = "healthy" if ok else "unhealthy"
    return JSONResponse({"status": status, "database": status}, status_code=200 if ok else 503)

from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Request
from fastapi.exceptions import RequestValidationError
from fastapi.encoders import jsonable_encoder
from fastapi.responses import JSONResponse

from .core.database import create_db
from .routers import health, items, stores, tags, users


@asynccontextmanager
async def lifespan(app: FastAPI):
    create_db()
    yield


app = FastAPI(title="Stores API", lifespan=lifespan)

app.include_router(users.router)
app.include_router(stores.router)
app.include_router(tags.router)
app.include_router(items.router)
app.include_router(health.router)


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    return JSONResponse({"message": exc.detail}, status_code=exc.status_code, headers=exc.headers)


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    return JSONResponse({"message": "Validation error", "errors": jsonable_encoder(exc.errors())}, status_code=422)

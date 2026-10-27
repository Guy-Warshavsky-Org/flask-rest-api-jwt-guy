from pydantic import BaseModel, ConfigDict, model_validator


class _Out(BaseModel):
    model_config = ConfigDict(from_attributes=True)


# Output models (mirror the Marshmallow auto-schemas)
class UserOut(_Out):
    id: int
    username: str


class StoreOut(_Out):
    id: int
    name: str
    user_id: int


class TagOut(_Out):
    id: int
    name: str
    store_id: int


class ItemOut(_Out):
    id: int
    name: str
    price: float
    store_id: int


# Request bodies
class Credentials(BaseModel):
    username: str
    password: str


class StoreIn(BaseModel):
    name: str


class TagIn(BaseModel):
    name: str


class ItemIn(BaseModel):
    name: str
    price: float
    store_id: int


class ItemUpdate(BaseModel):
    name: str | None = None
    price: float | None = None
    store_id: int | None = None

    @model_validator(mode="after")
    def _reject_explicit_null(self):
        for field in self.model_fields_set:
            if getattr(self, field) is None:
                raise ValueError(f"{field} must not be null")
        return self

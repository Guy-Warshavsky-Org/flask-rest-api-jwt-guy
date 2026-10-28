from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

_DEFAULT_SQLITE = "sqlite:///" + str(Path(__file__).resolve().parents[2] / "data-dev.sqlite")


class Settings(BaseSettings):
    """Environment-driven configuration (same variable names/defaults as the Flask app)."""

    model_config = SettingsConfigDict(env_file=".env", extra="ignore", populate_by_name=True)

    secret_key: str = Field("your-secret-key-change-me", validation_alias="SECRET_KEY")
    database_url: str | None = Field(None, validation_alias="DATABASE_URL")
    dev_database_url: str | None = Field(None, validation_alias="DEV_DATABASE_URL")
    access_token_expires_minutes: int = 15
    refresh_token_expires_days: int = 30

    @property
    def sqlalchemy_url(self) -> str:
        return self.database_url or self.dev_database_url or _DEFAULT_SQLITE


settings = Settings()

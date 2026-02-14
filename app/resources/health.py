from flask import Blueprint, jsonify
from sqlalchemy import text

from app import db

health_bp = Blueprint("health", __name__, url_prefix="/health")


@health_bp.route("/", methods=["GET"])
def health_check() -> tuple:
    """Lightweight liveness / readiness probe.

    Returns 200 when the API process is up **and** the database is reachable.
    Returns 503 if the database connection fails.
    """
    try:
        db.session.execute(text("SELECT 1"))
        db_status = "healthy"
        status_code = 200
    except Exception:
        db_status = "unhealthy"
        status_code = 503

    payload = {
        "status": "healthy" if status_code == 200 else "unhealthy",
        "database": db_status,
    }
    return jsonify(payload), status_code

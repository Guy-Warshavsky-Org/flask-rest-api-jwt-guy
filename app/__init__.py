from datetime import timedelta
from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from flask_jwt_extended import JWTManager
from config import config

db = SQLAlchemy()
jwt = JWTManager()

# In-memory token blacklist (for revocation)
jwt_token_blacklist = set()

def create_app(config_name='default'):
    app = Flask(__name__)
    app.config.from_object(config[config_name])
    app.config['JWT_SECRET_KEY'] = app.config['SECRET_KEY']
    app.config['JWT_ACCESS_TOKEN_EXPIRES'] = timedelta(minutes=15)
    app.config['JWT_REFRESH_TOKEN_EXPIRES'] = timedelta(days=30)

    db.init_app(app)
    jwt.init_app(app)

    # JWT callbacks for blacklist
    @jwt.token_in_blocklist_loader
    def check_if_token_revoked(jwt_header, jwt_payload):
        jti = jwt_payload['jti']
        return jti in jwt_token_blacklist

    from .models import create_db  # Import here to avoid circular
    with app.app_context():
        create_db()

    # Register blueprints
    from .resources.users import users_bp
    from .resources.stores import stores_bp
    from .resources.tags import tags_bp
    from .resources.items import items_bp
    from .resources.health import health_bp
    app.register_blueprint(users_bp)
    app.register_blueprint(stores_bp)
    app.register_blueprint(tags_bp)
    app.register_blueprint(items_bp)
    app.register_blueprint(health_bp)

    return app
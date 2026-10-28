# Flask REST API with JWT Authentication

A complete REST API built with FastAPI featuring JWT token authentication, user management, and CRUD operations for stores, items, and tags.

## Features

- 🔐 **JWT Authentication** - Secure access and refresh tokens
- 👤 **User Management** - Registration, login, and account management
- 🏪 **Store Management** - Create and manage multiple stores per user
- 📦 **Item Management** - Add products to stores with pricing
- 🏷️ **Tag System** - Categorize stores with custom tags
- 🗑️ **Cascade Deletion** - Automatic cleanup of related resources
- 🔒 **Authorization** - Users can only access their own resources

## Tech Stack

- **FastAPI / Uvicorn** - Web framework and ASGI server
- **SQLAlchemy** - ORM for database operations
- **PyJWT** - JWT token management
- **Pydantic v2** - Data validation and serialization
- **SQLite** - Development database (PostgreSQL ready for production)
- **Docker** - Containerization support

## Project Structure

```
├── app/
│   ├── main.py               # FastAPI app, error handlers, startup
│   ├── deps.py               # DB session + JWT bearer dependencies
│   ├── models.py             # SQLAlchemy 2.0 models
│   ├── schemas.py            # Pydantic schemas
│   ├── core/                 # settings, DB engine, JWT, blocklist
│   └── routers/              # users, stores, items, tags, health
├── tests/                    # pytest characterization tests
├── requirements.txt
├── Dockerfile
├── docker-compose.yml
└── INSOMNIA_TESTING_GUIDE_*.txt
```

## Installation

### Local Setup

1. **Clone the repository**

   ```bash
   git clone https://github.com/v1Rtu3-h05t/flask-rest-api-jwt.git
   cd flask-rest-api-jwt
   ```

2. **Create virtual environment**
   ```bash
   python -m venv venv
   ```

3. **Activate virtual environment**
   - Windows: `.\venv\Scripts\activate`
   - Mac/Linux: `source venv/bin/activate`

4. **Install dependencies**
   ```bash
   pip install -r requirements.txt
   ```

5. **Run the application**
   ```bash
   uvicorn app.main:app --port 5000
   ```

6. **Access the API**
   - Server runs at: `http://127.0.0.1:5000`

### Docker Setup

1. **Build and run with Docker Compose**
   ```bash
   docker-compose up --build
   ```

2. **Access the API**
   - Server runs at: `http://localhost:5000`
   - Uses PostgreSQL database

## API Endpoints

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/user/register` | None | Create new user account |
| POST | `/user/login` | None | Login and get tokens |
| POST | `/user/logout` | Access Token | Logout and revoke token |
| POST | `/user/refresh` | Refresh Token | Get new access token |

### Users

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/user/<id>` | Access Token | Get user details |
| DELETE | `/user/<id>` | Access Token | Delete user account |

### Stores

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/store/` | Access Token | Create new store |
| GET | `/store/<id>` | Access Token | Get store by ID |
| GET | `/store/s` | Access Token | Get all user's stores |
| DELETE | `/store/<id>` | Access Token | Delete store |

### Items

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/item/` | Access Token | Create new item |
| GET | `/item/<id>` | Access Token | Get item by ID |
| GET | `/item/s` | Access Token | Get all user's items |
| PUT | `/item/<id>` | Access Token | Update item |
| DELETE | `/item/<id>` | Access Token | Delete item |

### Tags

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/tag/store/<store_id>` | Access Token | Create tag for store |
| GET | `/tag/<id>` | Access Token | Get tag by ID |
| GET | `/tag/store/<store_id>/s` | Access Token | Get all tags in store |
| DELETE | `/tag/<id>` | Access Token | Delete tag |

## Authentication

This API uses JWT (JSON Web Tokens) for authentication:

- **Access Token**: Short-lived (15 minutes) - used for API requests
- **Refresh Token**: Long-lived (30 days) - used to get new access tokens

### Authorization Header Format

```
Authorization: Bearer <your_access_token>
```

## Example Usage

### 1. Register a User

```bash
POST http://127.0.0.1:5000/user/register
Content-Type: application/json

{
  "username": "johndoe",
  "password": "secure_password"
}
```

### 2. Login

```bash
POST http://127.0.0.1:5000/user/login
Content-Type: application/json

{
  "username": "johndoe",
  "password": "secure_password"
}
```

**Response:**
```json
{
  "access_token": "eyJ0eXAiOiJKV1Qi...",
  "refresh_token": "eyJ0eXAiOiJKV1Qi..."
}
```

### 3. Create a Store

```bash
POST http://127.0.0.1:5000/store/
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "My Store"
}
```

### 4. Create an Item

```bash
POST http://127.0.0.1:5000/item/
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Product Name",
  "price": 29.99,
  "store_id": 1
}
```

## Testing

### Using Insomnia/Postman

For **detailed step-by-step testing instructions**, see:
- `INSOMNIA_TESTING_GUIDE_PART1.txt` - Authentication, Stores, and Items
- `INSOMNIA_TESTING_GUIDE_PART2.txt` - Tags, Token Management, Troubleshooting, and Pro Tips

These guides include:
- Complete testing workflow
- Example requests and responses
- Common errors and solutions
- Best practices and pro tips

### Quick Test

1. Start the server
2. Register a user: `POST /user/register`
3. Login: `POST /user/login` → Save tokens
4. Create a store: `POST /store/` (use access token)
5. Create an item: `POST /item/` (use access token)

## Database Models

### User
- `id`: Integer (Primary Key)
- `username`: String (Unique)
- `password_hash`: String
- Relationships: One-to-Many with Stores

### Store
- `id`: Integer (Primary Key)
- `name`: String
- `user_id`: Integer (Foreign Key)
- Relationships: One-to-Many with Items and Tags

### Item
- `id`: Integer (Primary Key)
- `name`: String
- `price`: Float
- `store_id`: Integer (Foreign Key)

### Tag
- `id`: Integer (Primary Key)
- `name`: String
- `store_id`: Integer (Foreign Key)

## Configuration

### Environment Variables

- `SECRET_KEY`: Secret key for JWT tokens (default: 'your-secret-key-change-me')
- `DATABASE_URL`: Database connection string (PostgreSQL for production)
- `DEV_DATABASE_URL`: Development database (SQLite by default)

### Token Expiration

- Access Token: 15 minutes
- Refresh Token: 30 days

## Security Features

- ✅ Password hashing with Werkzeug
- ✅ JWT token authentication
- ✅ Token blacklist for logout
- ✅ User authorization checks
- ✅ Passwords never returned in responses
- ✅ Users can only access their own resources

## Troubleshooting

### Common Issues

**Connection Refused**
- Make sure server is running: `uvicorn app.main:app --port 5000`
- Check the URL: `http://127.0.0.1:5000`

**401 Unauthorized**
- Check token format: `Bearer <token>` (note the space)
- Token may have expired (15 min for access tokens)
- Use refresh token to get a new access token

**404 Not Found**
- Verify the resource ID exists
- Check endpoint spelling

For more detailed troubleshooting, see the testing guides.

## Future Enhancements

- [ ] Many-to-many relationship between Items and Tags
- [ ] Search and filter functionality
- [ ] Pagination for large datasets
- [ ] Rate limiting
- [ ] API documentation with Swagger/OpenAPI
- [ ] Email verification
- [ ] Password reset functionality
- [ ] Admin user roles

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Commit changes: `git commit -m 'Add feature'`
4. Push to branch: `git push origin feature-name`
5. Submit a pull request

## License

This project is open source and available under the MIT License.

## Author

Anthony Morales

---

**For complete testing instructions and examples, please refer to the included testing guides:**
- `INSOMNIA_TESTING_GUIDE_PART1.txt`
- `INSOMNIA_TESTING_GUIDE_PART2.txt`
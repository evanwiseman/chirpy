# chirpy
A simple http web server that mimics Twitter/X. Users are able to register an account, login, post chirps(tweets), see other users chirps, and modify their account information.

## Requirements
* go 1.22+
* sqlc 1.30+
* goose 3.25+
* psql 15+

## Installation
Clone the chripy repository to your desired location.

### SQLC
Instal SQLC:
```
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### Goose
Install goose:
```
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### PSQL
Install Postgres with v15+
Mac OS:
```
brew install postgresql@15
```

Linux/WSL (Debain):
```
sudo apt update
sudo apt install postgresql postgresql-contrib
```

Linux Only, set postgres password:
```
sudo passwd postgres
```

Start the postgres server in the background
* Mac `brew services start postgresql@15`
* Linux `sudo service postgresql start`

Enter the psql shell:
* Mac: `psql postgres`
* Linux: `sudo -u postgres psql`

Create a new database called chirpy:
```
CREATE DATABASE chirpy;
```

Connect to the database:
```
\c chirpy
```

Linux only set the database password:
```
ALTER USER postgres WITH PASSWORD 'postgres';
```

## Users
### Error Responses
Errors are always returned as JSON:  

```json
{
  "error": "description of what went wrong"
}
```
### POST `/api/users` – Register a User
Registers a new user with email and password.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Behavior:**
- Password is hashed before storing.
- Returns the created user.

**Response (201 Created):**
```json
{
  "id": "uuid-of-user",
  "created_at": "2025-10-02T12:34:56Z",
  "updated_at": "2025-10-02T12:34:56Z",
  "email": "user@example.com",
  "token": "",
  "refresh_token": ""
}
```

---

### PUT `/api/users` – Update User
Updates a user’s email and password. Requires a valid JWT access token.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "email": "newemail@example.com",
  "password": "newpassword456"
}
```

**Behavior:**
- Validates the access token.
- Hashes the new password.
- Updates user record in database.

**Response (200 OK):**
```json
{
  "id": "uuid-of-user",
  "created_at": "2025-10-02T12:34:56Z",
  "updated_at": "2025-10-02T12:34:56Z",
  "email": "newemail@example.com",
  "is_chirpy_red": false,
  "token": "<same access token>",
  "refresh_token": ""
}
```

---

### POST `/api/login` – Login User
Logs a user in with email and password, returning access and refresh tokens.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Behavior:**
- Validates user credentials.
- Generates a **JWT access token** (valid for 1 hour).
- Generates a **refresh token** (valid for 60 days).
- Stores refresh token in database.

**Response (200 OK):**
```json
{
  "id": "uuid-of-user",
  "created_at": "2025-10-02T12:34:56Z",
  "updated_at": "2025-10-02T12:34:56Z",
  "email": "user@example.com",
  "is_chirpy_red": false,
  "created_at": "2025-10-02T12:34:56Z",
  "updated_at": "2025-10-02T12:34:56Z",
  "token": "<jwt-access-token>",
  "refresh_token": "<refresh-token>"
}
```

---

## Chirps

### POST `/api/chirps` - Create Chrip

Requires JWT authentication. Body is limited to 140 characters, and banned words are censored.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "body": "Hello world!"
}
```

**Behavior:**
- Validates the acces token
- Validates the chirp
- Stores chirp in the database
- Returns the created chirp

**Response (201 Created):**
```json
{
  "id": "uuid-of-chirp",
  "created_at": "2025-10-02T12:34:56Z",
  "updated_at": "2025-10-02T12:34:56Z",
  "body": "Hello world!",
  "user_id": "uuid-of-user"
}
```

---

### Get All Chirps
`GET /api/chirps`

Retrieves all chirps. Supports optional query parameters:

- `author_id`: filter chirps by author
- `sort`: `asc` or `desc` for sorting by creation time

**Response (200 OK):**
```json
{
  {
    "id": "uuid-of-chirp-1",
    "created_at": "2025-10-01T10:00:00Z",
    "updated_at": "2025-10-01T10:00:00Z",
    "body": "First chirp!",
    "user_id": "uuid-of-user"
  },
  {
    "id": "uuid-of-chirp-2",
    "created_at": "2025-10-02T12:34:56Z",
    "updated_at": "2025-10-02T12:34:56Z",
    "body": "Another chirp!",
    "user_id": "uuid-of-user"
  }
}
```

---

### Get Chirp by ID
`GET /api/chirps/{chirpID}`

Retrieves a single chirp by its ID.

**Response (200 OK):**
```json
{
  "id": "uuid-of-chirp",
  "created_at": "2025-10-02T12:34:56Z",
  "updated_at": "2025-10-02T12:34:56Z",
  "body": "Hello world!",
  "user_id": "uuid-of-user"
}
```

---

### Delete Chirp by ID
`DELETE /api/chirps/{chirpID}`

Requires JWT authentication. Only the author of a chirp can delete it.

**Response (204 No Content):**
(no content)

## Webhooks

### Upgrade User to Chirpy Red
`POST /api/webhooks/upgrade-user-chirpy-red`

Upgrades a user to “Chirpy Red” when a valid event is received.

**Request Headers:**
- `Authorization`: API key

**Request Body:**
```json
{
  "data": {
    "user_id": "uuid-of-user"
  },
  "event": "user.upgraded"
}
```

**Behavior:**
1. Validates the API key.
2. Checks that the `event` field is `"user.upgraded"`.
3. Updates the user in the database to `is_chirpy_red = true`.
4. Returns `204 No Content` if successful or the event is ignored.

**Responses:**

**Success (user upgraded or event ignored):**
HTTP 204 No Content

**Invalid API Key:**
```json
{
  "error": "invalid api key: ..."
}
```

**Malformed Request Body:**
```json
{
  "error": "invalid format: ..."
}
```

**User Not Found:**
```json
{
  "error": "user not found: ..., uuid-of-user"
}
```

## Refresh Access Token

`POST /api/token/refresh`

Generates a new access token using a valid refresh token.

**Request Headers:**
- `Authorization`: Bearer `<refresh_token>`

**Behavior:**
1. Validates the presence of the refresh token in the database.
2. Checks if the refresh token has expired.
3. Checks if the refresh token has been revoked.
4. Generates a new JWT access token with the same expiration duration as the original refresh token.

**Request Body:**  
_None; token is passed via Authorization header._

**Response (200 OK):**
```json
{
  "token": "new-jwt-access-token"
}
```

## Revoke Access Token
`POST /api/token/refresh`

Revokes an access token.

**Request Headers:**
- `Authorization`: Bearer `<refresh_token>`

**Behavior:**
1. Attempt to remove the access token from the database

**Request Body:**  
_None; token is passed via Authorization header._

**Response (204 No Content)**

---

## Metrics

### MiddlewareMetricsInc
This middleware increments the file server hit counter for every request.

**Usage:**
Attach `MiddlewareMetricsInc` to routes you want to track hits for.

---

### Get Metrics

`GET /api/metrics`

Returns the total number of times the file server has been visited.

**Response (200 OK):**
Content-Type: `text/html`

```html
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited 123 times!</p>
  </body>
</html>
```

---

### Reset File Hits and Users

`POST /api/reset`

Resets the file server hit counter and attempts to reset the user database. **Only allowed in development environment.**

**Request Body:**  
_None_

**Behavior:**
1. Checks if `cfg.Platform` is `"dev"`. If not, returns `403 Forbidden`.
2. Resets the file server hit counter to `0`.
3. Calls `ResetUsers` on the database to reset all user data.

**Responses:**

- **Success (200 OK):**  
Content-Type: `text/plain; charset=utf-8`  
No response body.

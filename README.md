# ropasonline-user-app-backend

API REST de usuarios para la tienda en línea, desarrollada en Go y Gin con arquitectura hexagonal.

## Requisitos locales

- Go según la versión declarada en `go.mod`.
- MySQL accesible desde el entorno donde se ejecuta la API.

## Configuración

Crea un archivo `.env` local usando `.env.example` como referencia y coloca allí las credenciales de tu entorno. Configura `JWT_EXPIRATION_SECONDS` como un entero positivo en segundos. No compartas ni versionas el `.env` real.

Crea la base de datos configurada en `DB_NAME` (por defecto, `ropas_online_auth`) y aplica el script:

```sql
SOURCE migrations/001_create_users.sql;
```

El esquema incluye un UUID único, nombre y apellido para cada usuario. La tabla almacena el hash de la contraseña, nunca la contraseña en texto plano.

## Ejecutar

Desde WSL, en la raíz del proyecto:

```sh
go run .
```

El servidor usa `HTTP_ADDRESS` o, por defecto, `:8080`. `GET /ping` verifica que la API responde.

## Endpoints

- `POST /api/v1/auth/register`: crea una cuenta con `name`, `last_name`, `email` y `password`, y devuelve su UUID público.
- `POST /api/v1/auth/login`: valida credenciales y devuelve un JWT Bearer cuyo `sub` es el UUID público, más el perfil y la expiración configurada en `JWT_EXPIRATION_SECONDS`.
- `GET /api/v1/users/me`: obtiene el perfil asociado al token.
- `PUT /api/v1/users/me`: actualiza el nombre, apellido y email del usuario autenticado.
- `DELETE /api/v1/users/me`: elimina la cuenta del usuario autenticado.

Para las rutas de perfil, envía `Authorization: Bearer <token>`.

## Pruebas

```sh
go test ./...
```

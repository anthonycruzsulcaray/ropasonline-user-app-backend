# ropasonline-user-app-backend

API REST de autenticación para la tienda en línea, integrada con Amazon Cognito.

## Requisitos

- Go 1.23 o posterior.
- Un User Pool de Cognito con inicio de sesión por correo y verificación de correo habilitada.
- Un App Client que habilite el flujo `USER_PASSWORD_AUTH`. Si el App Client tiene un secreto, configúralo en `COGNITO_CLIENT_SECRET`.
- Credenciales AWS disponibles mediante el proveedor estándar del SDK (por ejemplo, un rol IAM en EC2). Para crear cuentas, confirmar cuentas e iniciar sesión, el App Client debe permitir esas operaciones en el User Pool.

Cognito envía el código de confirmación de correo con la longitud definida por el propio servicio; la aplicación no genera ni modifica ese código.

## Configuración y ejecución

Copia `.env.example` a `.env` y completa `AWS_REGION` y `COGNITO_CLIENT_ID`. `COGNITO_CLIENT_SECRET` es opcional; déjalo vacío para un App Client sin secreto. La aplicación busca `.env` en el directorio actual y sus directorios padres; las variables de entorno ya definidas y no vacías tienen prioridad. No guardes `.env` en Git.

```powershell
go mod tidy
go run ./cmd/api
```

En producción, suministra estas variables mediante el entorno y credenciales AWS de ejecución; no se requieren claves AWS estáticas en el archivo `.env`.

## Endpoints

- `POST /api/auth/register` — cuerpo: `{"email":"usuario@ejemplo.com","password":"..."}`. Registra la cuenta y Cognito envía el código por correo. La respuesta indica `confirmation_required`.
- `POST /api/auth/confirm` — cuerpo: `{"email":"usuario@ejemplo.com","code":"..."}`. Confirma la cuenta con el código que envió Cognito.
- `POST /api/auth/resend-confirmation` — cuerpo: `{"email":"usuario@ejemplo.com"}`. Reenvía el código a una cuenta pendiente de confirmación. La respuesta puede incluir el canal y destino enmascarado que informa Cognito.
- `POST /api/auth/login` — cuerpo: `{"email":"usuario@ejemplo.com","password":"..."}`. Devuelve los tokens de Cognito (`access_token`, `id_token`, `refresh_token`, `token_type`, `expires_in`).
- `GET /healthz` — estado básico del proceso.

La confirmación no devuelve tokens: Cognito entrega los tokens al iniciar sesión después de confirmar la cuenta. Las contraseñas y códigos se transmiten únicamente en el cuerpo JSON y no se devuelven en las respuestas.

Ejecuta las pruebas con `go test ./...`.

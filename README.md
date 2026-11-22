# cookie-lab

Mínimo para experimentar con cookies HTTP entre un back en Go (Gin) y un front en React (Vite). El back setea una cookie `session` y en la request siguiente el navegador la manda solo. Tiene dos modos:

|                       | A. Same-origin                                  | B. Cross-site                                              |
| --------------------- | ----------------------------------------------- | ---------------------------------------------------------- |
| Servicios             | uno solo: Go sirve la SPA (`go:embed`) y `/api` | front (static site) y API separados                        |
| Cookie                | `HttpOnly; Secure; SameSite=Lax; Path=/`        | `HttpOnly; Secure; SameSite=None; Path=/`                  |
| CORS                  | no hace falta                                   | origen exacto + `Access-Control-Allow-Credentials: true`   |
| Env del back          | `COOKIE_MODE=same-site` (default)               | `COOKIE_MODE=cross-site`, `ALLOWED_ORIGIN=https://<front>` |
| Env del front (build) | `VITE_API_URL` vacío                            | `VITE_API_URL=https://<api>`                               |

`onrender.com` está en la [Public Suffix List](https://publicsuffix.org/), así que `cookie-lab-front.onrender.com` y `cookie-lab-api.onrender.com` son **sitios distintos**: para el navegador, la cookie de B es una cookie de terceros.

## API

- `POST /api/login`: genera un valor random y hace `Set-Cookie: session=<valor>`. Responde `{ set, value }`.
- `GET /api/me`: responde `{ received, value, rawCookieHeader }` con lo que llegó realmente.
- `POST /api/logout`: borra la cookie con `Max-Age=0` y los mismos atributos.

Variables de entorno: `PORT` (default `8080`), `COOKIE_MODE` (`same-site` | `cross-site`), `ALLOWED_ORIGIN` (solo cross-site; `scheme://host[:port]` exacto, sin barra final) y `GIN_MODE` (default `release`).

## Local

Requisitos: Go (versión de `backend/go.mod`) y Node 24.

```sh
# Opción 1: dev con hot reload (el proxy de Vite manda /api al back, así que es same-origin)
make run                         # back en :8080
cd frontend && npm install && npm run dev   # front en http://localhost:5173

# Opción 2: igual que en producción (modo A), un solo binario
make web && make run             # http://localhost:8080
```

Chrome y Firefox aceptan cookies `Secure` en `http://localhost`. **Safari no**: ahí el login responde OK, pero la cookie no se guarda. Probá en Chrome o Firefox, o usá el deploy.

Calidad: `make test` corre los tests con race y coverage, `make lint` corre golangci-lint (exige la última release) y `make lint-install` la instala.

## Deploy en Render (gratis)

1. Subí el repo a GitHub.
2. En Render: **New > Blueprint**, conectá el repo y aplicá `render.yaml`. Se crean tres servicios en plan free: `cookie-lab` (A), `cookie-lab-api` y `cookie-lab-front` (B).
3. Revisá las URLs que asignó Render. Si algún nombre estaba ocupado, Render le agrega un sufijo. En ese caso corregí `ALLOWED_ORIGIN` en `cookie-lab-api` y `VITE_API_URL` en `cookie-lab-front`, y redeployá el front: `VITE_API_URL` se resuelve en build time. Van como literales porque `fromService` del blueprint solo expone el host interno, no la URL pública.

Límites del free tier: los web services se duermen después de 15 min sin tráfico, y la primera request tarda alrededor de un minuto en despertarlos. Las 750 horas de instancia por mes se comparten entre todos los servicios free del workspace (los dormidos no consumen), y si se agotan se suspenden hasta el mes siguiente. Además, el filesystem es efímero. El static site no se duerme.

## Qué mirar

Abrí DevTools:

- **Network**: en la request de login, el header de respuesta `Set-Cookie`. En la de me, el header de request `Cookie`.
- **Application > Cookies** (Firefox: _Storage_; Safari: _Storage > Cookies_): la cookie con sus atributos.

Flujo: **Login → Me → Logout → Me**. Si la cookie viaja, el segundo paso muestra `received: true` y el mismo `value`; después del logout, `received: false`.

### A (same-origin): `https://cookie-lab.onrender.com`

Funciona igual en Chrome, Firefox y Safari. No hay preflight ni headers CORS. Como la cookie es `HttpOnly`, `document.cookie` no la ve.

### B (cross-site): `https://cookie-lab-front.onrender.com`

- Los `POST` con `Content-Type: application/json` disparan un **preflight `OPTIONS`**. Se ve en Network con `Access-Control-Allow-Origin` (el origen exacto, nunca `*`) y `Access-Control-Allow-Credentials: true`.
- **Chrome** (ventana normal, configuración por defecto): funciona. La cookie aparece bajo el dominio de la API, no del front. En incógnito, Chrome bloquea las cookies de terceros por defecto y falla: Network marca el `Set-Cookie` como bloqueado (hay un ícono de advertencia con el motivo).
- **Safari**: ITP bloquea todas las cookies de terceros, así que el login responde 200 con el valor pero `/me` devuelve `received: false`.
- **Firefox**: con Total Cookie Protection (el default), la cookie se guarda **particionada** bajo el sitio del front. Puede parecer que funciona, pero si abrís la API directo en otra pestaña no la vas a ver. Con bloqueo estricto de cookies cross-site falla igual que Safari.

El comportamiento cambia entre versiones y configuraciones de cada navegador. Por eso lo que vale es lo que muestra Network en el momento.

Si `/me` responde `received: false` en B, el back está bien: el navegador decidió no guardar o no mandar la cookie. Si en cambio el fetch tira `TypeError: Failed to fetch`, es CORS (revisá `ALLOWED_ORIGIN`), y el detalle está en la consola.

`cookie-lab-api` usa la misma imagen que A, así que en `https://cookie-lab-api.onrender.com/` también se sirve la SPA embebida. Pero ahí es same-origin con la API, no es el experimento B: para B, siempre entrá por el front.

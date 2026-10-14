# Painel de administração

Painel em Next.js (App Router) e Tailwind CSS para a API do Email Pixel Tracker.

O navegador conversa só com o servidor do Next.js. O token da API fica em um cookie
`httpOnly`, e todas as chamadas à API são feitas pelo servidor: o token e o endereço
interno da API nunca chegam ao navegador. Telas e botões aparecem conforme
`permissions` de `GET /api/auth/me`, mas a API continua validando cada ação.

## Desenvolvimento

Com a API rodando (por exemplo com `build.bat`/`build.sh` na raiz do projeto):

```bash
cp .env.example .env.local   # ajuste se a API não estiver em localhost:8080
npm install
npm run dev                  # http://localhost:3000
```

Entre com o SuperAdmin configurado no `.env` da API (`SUPERADMIN_USERNAME`/`SUPERADMIN_PASSWORD`).

## Variáveis de ambiente

| Variável | Padrão | Uso |
|---|---|---|
| `API_URL` | `http://localhost:8080` | Endereço da API visto pelo servidor do Next.js. |
| `PIXEL_BASE_URL` | `http://localhost:8080` | Endereço público da API, usado no snippet do pixel. |
| `COOKIE_SECURE` | `true` em produção | Cookie de sessão só em HTTPS. Use `false` para servir em HTTP fora de `localhost`. |

## reCAPTCHA

O reCAPTCHA v3 do login é configurado na API (`RECAPTCHA_*` no `.env` da raiz), não aqui.
A tela de login pergunta à API (`GET /api/auth/recaptcha`) se ele está ativo e, se estiver,
carrega o script do Google e envia o token no login. Com ele ativo, o login exige JavaScript.

## Docker

```bash
# Só o painel, apontando para uma API já rodando nesta máquina
docker compose up --build

# API + banco + painel (a partir da raiz do projeto)
docker compose -f docker-compose.postgres.yml -f docker-compose.admin.yml up --build
```

# Email Pixel Tracker

API multi-tenant em Go para medir a abertura de e-mails. Cada campanha tem um pixel
(uma imagem PNG de 1×1). Quando o destinatário abre o e-mail, o cliente de e-mail baixa
a imagem e a API registra a visualização. Os relatórios mostram o total e as visualizações
únicas por dia, por cliente de e-mail, por sistema operacional e por dispositivo, e podem
ser exportados em Excel.

Junto com a API há um [painel de administração](admin/README.md) em Next.js.

## Sumário

- [Como funciona](#como-funciona)
- [Rodando localmente](#rodando-localmente)
- [Rodando com Docker](#rodando-com-docker)
- [Configuração](#configuração)
- [Usando a API](#usando-a-api)
- [Observabilidade](#observabilidade)
- [Desenvolvimento](#desenvolvimento)

## Como funciona

1. Um **SuperAdmin** cadastra os **tenants** (os clientes da plataforma) e os usuários deles.
2. Os usuários do tenant criam **campanhas**. Cada campanha tem uma data de expiração e a cor do pixel.
3. O pixel vai no HTML do e-mail:

   ```html
   <img src="https://sua-api/pixel/<id-da-campanha>?externalUserId=<id-do-destinatario>" width="1" height="1" alt="">
   ```

4. Cada abertura gera uma visualização, com data, IP, user agent e o `externalUserId`, quando informado.
5. Os **relatórios** contam as visualizações:
   - Uma visualização **única** é um `externalUserId` distinto ou, nas visualizações sem ele, um IP distinto.
   - Sem `externalUserId`, destinatários atrás do mesmo IP contam como um só.

O pixel só conta visualizações de campanhas ativas, dentro da validade e de tenants
ativos. Nos outros casos a resposta é 404, sem corpo e sem registro. A imagem é servida
com cache desabilitado, para que cada abertura chegue até a API.

### Perfis de acesso

Cada usuário tem uma única role:

| Role | Tenants | Usuários | Campanhas | Relatórios |
|---|---|---|---|---|
| SuperAdmin | cadastra e edita | cadastra e edita | cadastra e edita (impersonando) | lê (impersonando) |
| Admin | lê o próprio | cadastra e edita | cadastra e edita | lê |
| ContentManager | lê o próprio | — | cadastra e edita | lê |
| API | lê o próprio | — | lê | lê |

- O SuperAdmin não pertence a nenhum tenant. Para ver ou alterar os dados de um tenant, ele usa
  `POST /api/auth/impersonate`. Um tenant inativo só pode ser acessado em modo de leitura.
- A role **API** é somente leitura e serve para integrações.
- A cada requisição, o usuário e o tenant são relidos do banco. Desativar um usuário, trocar a
  role dele ou desativar o tenant tem efeito imediato, mesmo que o token ainda seja válido.

## Rodando localmente

Requisitos: Go 1.26 ou mais recente, e um banco de dados: PostgreSQL, MySQL, SQL Server ou MongoDB.

```bash
cp .env.example .env   # ajuste o banco, JWT_SECRET e SUPERADMIN_PASSWORD
./build.sh             # Linux/macOS
build.bat              # Windows
```

Os scripts compilam a API em `email-tracker` (ou `email-tracker.exe`, no Windows) na
raiz do projeto e a executam. Argumentos extras são repassados ao executável.

Na primeira execução:
- As tabelas são criadas automaticamente, ou as coleções e os índices no MongoDB.
- O SuperAdmin definido em `SUPERADMIN_USERNAME` e `SUPERADMIN_PASSWORD` é criado.

Com `DOCS_ENABLED=true`, a documentação interativa (Swagger) fica em
`http://localhost:8080/docs/`.

## Rodando com Docker

Há um arquivo compose por banco. Cada um sobe o banco e a API, que só inicia depois que o
banco estiver saudável. `JWT_SECRET` e `SUPERADMIN_PASSWORD` são lidos do `.env` da raiz e
são obrigatórios.

```bash
docker compose -f docker-compose.postgres.yml up --build
docker compose -f docker-compose.mysql.yml up --build
docker compose -f docker-compose.sqlserver.yml up --build
docker compose -f docker-compose.mongodb.yml up --build
```

Arquivos opcionais, combináveis com qualquer um dos anteriores:

```bash
# Painel de administração em http://localhost:3000
docker compose -f docker-compose.postgres.yml -f docker-compose.admin.yml up --build

# Jaeger (traces) em http://localhost:16686 e Prometheus (métricas) em http://localhost:9090
docker compose -f docker-compose.postgres.yml -f docker-compose.observability.yml up --build
```

- A API fica em `http://localhost:8080`; a porta muda com `API_PORT`.
- Nos composes, a documentação vem ligada e é servida na raiz: `http://localhost:8080/`.
- A imagem final é distroless e roda sem root. O health check é o próprio binário, com
  `-healthcheck`.

## Configuração

A configuração vem de variáveis de ambiente ou do arquivo `.env`. Variáveis já definidas no
ambiente têm prioridade. O [.env.example](.env.example) traz todas, comentadas.

| Variável | Padrão | Descrição |
|---|---|---|
| `DB_DRIVER` | — | `postgres`, `mysql`, `sqlserver` ou `mongodb`. |
| `DB_DSN` | — | String de conexão. No MySQL, `parseTime=true` é obrigatório. |
| `DB_NAME` | — | Nome do banco; usado só pelo MongoDB. |
| `HTTP_ADDR` | `:8080` | Endereço da API. |
| `TRUSTED_PROXIES` | vazio | IPs ou CIDRs dos proxies reversos autorizados a enviar `X-Forwarded-For`, separados por vírgula. **Obrigatório atrás de um proxy**; sem ele, todas as visualizações recebem o IP do proxy. |
| `JWT_SECRET` | — | Segredo dos tokens, com pelo menos 32 bytes. |
| `JWT_TTL` | `1h` | Validade dos tokens. |
| `BCRYPT_COST` | `12` | Custo do hash das senhas (de 4 a 31). |
| `SUPERADMIN_USERNAME` | — | SuperAdmin criado na inicialização se ainda não existir. |
| `SUPERADMIN_NAME` | `Super Admin` | Nome de exibição do SuperAdmin. |
| `SUPERADMIN_PASSWORD` | — | Senha do SuperAdmin, usada só na criação. |
| `DOCS_ENABLED` | `false` | Liga o Swagger UI e o `openapi.yaml`. |
| `DOCS_PATH` | vazio | Caminho da documentação; vazio serve na raiz. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error`. |
| `LOG_FORMAT` | `json` | `json` ou `text`. |
| `METRICS_ADDR` | vazio | Endereço das métricas Prometheus (ex.: `:9464`); vazio desliga. Não exponha publicamente. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | vazio | Endpoint OTLP/HTTP dos traces (ex.: `http://localhost:4318`); vazio desliga. |
| `RECAPTCHA_ENABLED` | `false` | Exige reCAPTCHA v3 no login. |
| `RECAPTCHA_SITE_KEY` / `RECAPTCHA_SECRET_KEY` | — | Chaves do reCAPTCHA, obrigatórias quando ele está ativo. |
| `RECAPTCHA_MIN_SCORE` | `0.5` | Nota mínima aceita, de 0.0 (robô) a 1.0 (pessoa). |

## Usando a API

A referência completa está no [openapi.yaml](api/docs/openapi.yaml), também servido pelo
Swagger UI. Resumo das rotas:

| Rota | Descrição |
|---|---|
| `POST /api/auth/login` | Login: devolve um token JWT. |
| `GET /api/auth/recaptcha` | Informa se o login exige reCAPTCHA (pública). |
| `GET /api/auth/me` | Usuário atual, tenant e permissões. |
| `POST /api/auth/impersonate` / `stop-impersonation` | SuperAdmin entra em um tenant ou sai dele. |
| `GET /api/roles` | Roles, indicando quais o usuário atual pode atribuir. |
| `/api/tenants` | Lista, cadastro e edição de tenants. |
| `/api/users` | Lista, cadastro e edição de usuários. |
| `/api/campaigns` | Lista, cadastro e edição de campanhas. |
| `GET /api/campaigns/{id}/views` | Visualizações, paginadas. |
| `GET /api/campaigns/{id}/reports/summary` | Total e visualizações únicas. |
| `GET /api/campaigns/{id}/reports/daily` | Por dia (UTC), com até 366 dias. |
| `GET /api/campaigns/{id}/reports/user-agents` | Por cliente de e-mail, sistema operacional, dispositivo e user agent. |
| `GET /api/campaigns/{id}/reports/export` | Todos os relatórios em uma planilha `.xlsx`. |
| `GET /pixel/{campaignId}` | O pixel (pública). |
| `GET /health/live` / `GET /health/ready` | Health checks: o `ready` testa o banco. |

Exemplo:

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"superadmin","password":"sua-senha"}' | jq -r .token)

curl -s localhost:8080/api/auth/me -H "Authorization: Bearer $TOKEN"
```

Detalhes úteis:
- Os relatórios aceitam `fromUtc` e `toUtc` no formato RFC 3339. O período é `[fromUtc, toUtc)`.
- As listas aceitam `page` e `pageSize`: 20 itens por página por padrão, no máximo 100.
- Erros vêm no formato `{"error": "...", "fields": [...]}`. Erros internos (500) trazem
  também um `requestId`, para encontrar a requisição nos logs.
- A classificação por user agent conhece os casos de e-mail:
  - O Gmail e o Yahoo Mail buscam as imagens por um proxy, que esconde o sistema e o
    dispositivo de quem abriu.
  - O Apple Mail é reconhecido mesmo sem se identificar como navegador.

## Observabilidade

- **Logs estruturados** (`log/slog`): uma linha por requisição com rota, status, duração
  e usuário. Logins, impersonações e falhas também são registrados.
- **Request ID:** toda resposta traz `X-Request-ID`, que também pode ser enviado na requisição.
  Ele aparece em todos os logs da requisição.
- **Métricas Prometheus** em `METRICS_ADDR`, com prefixo `email_tracker_`: requisições HTTP,
  pixels por resultado, logins, impersonações, duração das operações no banco e métricas do
  runtime do Go.
- **Traces OpenTelemetry**: HTTP e banco de dados, exportados via OTLP quando
  `OTEL_EXPORTER_OTLP_ENDPOINT` está definido. O header `traceparent` é aceito.

## Desenvolvimento

```bash
go test ./...
go vet ./...
```

Os testes usam um repositório em memória e não precisam de banco. Os testes da API também
validam cada resposta contra o `openapi.yaml`: ao mudar uma rota, atualize a especificação.

Estrutura:

| Pasta | Conteúdo |
|---|---|
| [cmd/api](cmd/api) | `main`: configuração, servidores e encerramento. |
| [api](api) | Rotas HTTP (gin), DTOs, exportação Excel e a documentação ([api/docs](api/docs)). |
| [service](service) | Regras de negócio e permissões. |
| [domain](domain) | Entidades. |
| [repository](repository) | Interfaces de acesso a dados e as implementações SQL (GORM), MongoDB e em memória. |
| [auth](auth) | O usuário autenticado (principal) no contexto. |
| [useragent](useragent) | Classificação de user agents. |
| [recaptcha](recaptcha) | Verificação do reCAPTCHA v3. |
| [telemetry](telemetry) | Logs, métricas e traces. |
| [admin](admin) | Painel de administração (Next.js), com Dockerfile próprio. |

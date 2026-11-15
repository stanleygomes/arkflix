# API do arkflix (`/arkflix/api`)

API própria do arkflix-server, além das três compatíveis. O app arkflix usa
pra administrar o servidor e pras listas. JSON em camelCase, datas RFC 3339
(UTC; `null` = nunca/sem data). Erro: status HTTP + `{"error": "mensagem em
português"}`.

## Login

Qualquer token que o servidor já emite vale aqui:

- `Authorization: MediaBrowser ..., Token="<token>"` ou `X-Emby-Token` (token do Jellyfin)
- `Authorization: Bearer <token>` (access token ou client token `rmm_...` do RomM)
- `?api_key=<token>` (pra `<img src>`; qualquer um dos acima)

Sem token válido: `401`. Rota de admin com usuário comum: `403`.

## Público

`GET /arkflix/api/info` (sem login)

```json
{ "product": "arkflix-server", "name": "arkflix", "version": "1.2.3",
  "writable": false,
  "features": ["admin", "lists", "upload", "metadata", "import"] }
```

O app chama isso no mesmo endereço do Jellyfin/RomM pra saber se o servidor é
um arkflix-server (e mostrar o painel).

## Eu

`GET /arkflix/api/me` → `{ "id": "32hex", "rid": 1, "name": "stanley", "admin": true }`

## Admin (só `admin: true`)

`GET /arkflix/api/status`

```json
{
  "version": "1.2.3", "writable": false, "startedAt": "...",
  "libraries": [
    { "id": "movies", "name": "Filmes", "path": "/var/media/Movies", "count": 120, "online": true },
    { "id": "shows",  "name": "Séries", "path": "...", "count": 14, "episodes": 420, "online": true },
    { "id": "music",  "name": "Músicas", "path": "...", "count": 900, "albums": 80, "artists": 60, "online": true },
    { "id": "games",  "name": "Jogos", "path": "...", "count": 300, "platforms": 12, "online": true }
  ],
  "scan": { "running": false, "startedAt": "...", "finishedAt": "...", "count": 1500, "error": "" },
  "metadata": { "enabled": true, "tmdb": false, "running": false, "pending": 12, "error": "" },
  "sessions": [ { "user": "stanley", "client": "Infuse", "device": "Apple TV", "lastSeen": "..." } ]
}
```

`count` é o número de itens principais (filmes, séries, faixas, ROMs).
`online: false` = pasta configurada mas não acessível (disco desmontado).

`POST /arkflix/api/scan` → `202 {}` (dispara em segundo plano; acompanhar
pelo `status`).

### Usuários

- `GET /arkflix/api/users` → `[{ "id", "rid", "name", "admin", "createdAt", "lastLogin" }]`
- `POST /arkflix/api/users` `{ "name", "password", "admin" }` → `201` usuário
- `PATCH /arkflix/api/users/{id}` `{ "password"?, "admin"? }` → usuário (tirar o
  admin do último administrador: `409`)
- `DELETE /arkflix/api/users/{id}` → `204` (não apaga a si mesmo nem o último admin: `409`)

### Enviar arquivo (precisa de `writable`)

`POST /arkflix/api/upload` — `multipart/form-data`, um ou mais campos `file`,
e os campos de destino:

| `kind` | Campos | Vai pra |
| --- | --- | --- |
| `movie` | `title`?, `year`? | `<movies>/<Título (Ano)>/<Título (Ano)>.<ext>` (sem título: nome do arquivo) |
| `episode` | `series`, `season`, `episode`?, `title`? | `<shows>/<Série>/Season NN/<Série> - SNNEMM[ - Título].<ext>` (sem `episode`: nome do arquivo dentro da temporada) |
| `music` | `folder`? | `<music>/<folder ou "Avulsas">/<arquivo>` |
| `rom` | `platform` (slug) | `<games>/roms/<slug>/<arquivo>` |
| `bios` | `platform` (slug) | `<games>/bios/<slug>/<arquivo>` |
| `subtitle` | `item` (id do filme/episódio), `lang`? | ao lado do vídeo, `<base>[.<lang>].<ext>` |

Mande os campos **antes** dos arquivos: aí cada arquivo vai direto pro
destino, sem passar pela memória (tamanho livre). Arquivo que chega antes
de `kind` espera em `<data>/uploads` e é movido no fim. `folder` da música
pode ter subpasta (`Artista/Álbum`). `movie`/`episode` só aceitam vídeo;
`subtitle`, `srt|vtt|ass|ssa|sub`.

Grava em `.<nome>.part` (oculto, o scan pula) e renomeia no fim; não
sobrescreve (nome repetido: `409`). Nome de arquivo e campos limpos (sem `/`,
`..`, controle). Campo faltando ou inválido: `400`. Depois dispara o scan.
Resposta `201 { "saved": ["Movies/Nome (2020)/Nome (2020).mkv"] }`.
Somente leitura: `403 { "error": "servidor em modo só leitura" }`.

### Esconder item (precisa de `writable`)

`POST /arkflix/api/items/{id}/hide` → move o arquivo (ou a pasta da ROM
multidisco, a temporada, a série) pra `<hidden>/<pasta da biblioteca>/<mesmo
caminho relativo>` (ex.: `escondidos/Movies/Nome (2020)/...`) e tira do
acervo. `204`. Desfazer = mover de volta à mão.

- Filme na pasta dele vai com a pasta (pôster, legendas); arquivo solto vai
  com o que tiver o mesmo nome ao lado (legenda, `.nfo`). Álbum = as faixas.
- Artista e console: `400`. Já existe em escondidos ou sem `--hidden`: `409`.
- `{id}` aceita `rom:<rid>`.

### Metadados

`POST /arkflix/api/items/{id}/refresh` → busca de novo os metadados do item
(`202`; por enquanto roda o `metadata` em segundo plano, que completa o que
falta).

## Listas (qualquer usuário, as dele)

Uma lista guarda itens de qualquer tipo (filme, série, episódio, ROM, álbum,
faixa). Aparecem também como Playlist no Jellyfin, coleção no RomM (só as
ROMs) e playlist no Subsonic (só as faixas).

- `GET /arkflix/api/lists` → `[{ "id": "32hex", "rid": 3, "name", "count", "kinds": ["movie","rom"], "createdAt", "updatedAt", "cover": "<item id ou vazio>" }]`
- `POST /arkflix/api/lists` `{ "name" }` → `201` lista
- `GET /arkflix/api/lists/{id}` → lista + `"items": [Item]`
- `PATCH /arkflix/api/lists/{id}` `{ "name" }` → lista
- `DELETE /arkflix/api/lists/{id}` → `204`
- `POST /arkflix/api/lists/{id}/items` `{ "ids": ["32hex" | "rom:<rid>"] }` → lista (repetido é ignorado; item que não existe ou de outro tipo: `400`)
- `DELETE /arkflix/api/lists/{id}/items/{itemId}` → `204` (aceita `rom:<rid>`)
- `PUT /arkflix/api/lists/{id}/order` `{ "ids": [...] }` → lista + `items` (os que não vieram ficam no fim)

Lista de outro usuário: `404`. `count`/`kinds`/`items` contam só o que ainda
está no acervo. Nos outros clientes aparecem as listas do usuário e as
públicas (Subsonic `public=true`) que têm algo daquele cliente ou estão
vazias.

`Item` = `{ "id": "32hex", "rid": 42, "kind": "movie|series|episode|rom|album|track", "name", "year", "platform"? (slug), "seriesName"?, "image": "/arkflix/api/items/{id}/image" ou "" }`.

`GET /arkflix/api/items/{id}/image?maxWidth=&maxHeight=` → a capa (mesma do
Jellyfin/RomM: a do item, da série, do álbum ou um quadro do vídeo), com
login. Sem imagem: `404`.

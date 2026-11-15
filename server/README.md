# arkflix-server

Um servidor só no lugar do **Jellyfin** (filmes e séries), do **Navidrome**
(música) e do **RomM** (jogos). Ele fala a API dos três, então os clientes que
já existem continuam funcionando sem mudar nada: Jellyfin no celular/TV,
Symfonium/play:Sub/Feishin na música, Argosy/muOS e o próprio app arkflix nos
jogos.

Escopo: **consumir** o que já está no disco. Você continua mandando mídia pras
pastas (o `noah_dev` faz isso); o servidor escaneia, lista, toca, baixa e
sincroniza saves. O que falta no disco (sinopse, pôster, capa) vem da
internet (veja Metadados).

## Convivência com os três atuais

Ele lê as **mesmas pastas, só leitura**, e escreve só na pasta de dados dele
(`--data`). Dá pra rodar do lado do Jellyfin/Navidrome/RomM até o phase-out,
apontando um domínio novo pra porta 8097 (veja `compose.example.yml`). No dia
da troca, o Caddy aponta os três domínios de hoje pra cá.

| Rota | API |
| --- | --- |
| `/rest/*` | Subsonic / OpenSubsonic (Navidrome) |
| `/api/*`, `/assets/romm/*` | RomM |
| `/health` | saúde, sem login |
| o resto | Jellyfin |

## Pastas (as mesmas de hoje)

```
--movies /var/media/Movies    Nome (Ano)/Nome (Ano).mkv, poster.jpg, .srt ao lado
--shows  /var/media/Shows     Série/Season 01/Série - S01E02 - Título.mkv
--music  /var/media/musicas   tags (ID3/Vorbis); sem tag, "Artista - Título"
--games  /var/media/jogos     roms/<console>/... e bios/<console>/... (layout do RomM)
```

O scan roda ao subir e de hora em hora; só relê arquivo que mudou
(tamanho/mtime). Os clientes também disparam scan (`startScan`, tarefas do RomM).

## Metadados

Depois de cada scan, em segundo plano, o servidor completa pela internet o
que o disco não trouxe. Só item com dado faltando; o que achou não é buscado
de novo (revisto no mês se ainda faltar algo) e o que falhou só é tentado no
outro dia. O painel de admin força um item (`Refresh`).

| Provedor | Itens | O que preenche |
| --- | --- | --- |
| TMDB (precisa de `--tmdb-key`) | filmes, séries, temporadas, episódios | sinopse (cai pro inglês se vazia), gêneros no idioma, estreia/ano, pôster, fundo, logo, still e nome do episódio sem título |
| libretro thumbnails | ROMs | capa (`Named_Boxarts`) e fundo (`Named_Snaps`/`Named_Titles`) pelo nome No-Intro/Redump |
| MusicBrainz + Cover Art Archive | álbuns sem capa | capa da frente |

- `--metadata=false` (`ARKFLIX_SERVER_METADATA=0`) desliga toda busca na internet.
- `--tmdb-key` aceita a chave v3 ou o token v4 (Bearer). Sem chave, filmes e
  séries ficam só com o disco; jogos e música continuam.
- `--language` (padrão `pt-BR`) é o idioma de sinopses, gêneros e logos.
- **O disco ganha sempre**: `.nfo`, imagem ao lado e tag nunca são trocados.
  O que veio da internet fica anotado em `extra.meta` do item (fonte, data,
  campos) e sobrevive aos rescans; se o disco passar a ter o campo, ele ganha.
- Imagens baixadas vão pra `<data>/metadata/<id do item>/` (`poster`,
  `backdrop`, `logo`, `still`; `.jpg` ou `.png`); item que sai do acervo
  leva a pasta junto. Ids e extras: `tmdb_id`, `imdb_id`, `rating`,
  `tagline`, `mbid` no `extra` do item.

## Login

Um usuário vale pras três APIs. A senha fica em bcrypt e também cifrada
(AES-GCM, chave em `<data>/secret.key`), porque o token do Subsonic é
`md5(senha+sal)` e só confere com a senha em mãos, como no Navidrome.

```sh
make server
build/arkflix-server --data /tmp/ark user add stanley --admin   # senha pela entrada
build/arkflix-server --data /tmp/ark --movies ... --music ... --games ...
```

Na 1ª subida em contêiner: `ARKFLIX_SERVER_ADMIN_USER`/`_PASSWORD` criam o admin.
Toda flag tem um `ARKFLIX_SERVER_*` equivalente (`--addr` → `ARKFLIX_SERVER_ADDR`).

Estado compartilhado: estrela no Subsonic = favorito no Jellyfin; onde parou e
"assistido/ouvido" valem pra todos os clientes.

## O que cada API cobre

- **Jellyfin** (10.10.x): login por aparelho, bibliotecas, itens com filtros,
  continuar/próximo/recentes, temporadas e episódios, imagens (com quadro do
  vídeo quando não há pôster), PlaybackInfo pelo perfil do aparelho (direto ou
  HLS h264/aac pelo ffmpeg), legendas externas e embutidas em VTT, progresso,
  favorito/assistido, música (Finamp), websocket de keepalive.
- **Subsonic/OpenSubsonic** (1.16.1): três jeitos de login (senha, token,
  apiKey), XML/JSON/JSONP, navegação por pasta e por ID3, listas, busca
  (vazia = tudo, pra sincronizar), stream com transcodificação, capa, letras
  (`.lrc`), estrela/nota/scrobble, playlists, fila e marcadores.
- **RomM** (anuncia 4.8.0): OAuth2 (senha/refresh), Basic, client tokens
  `rmm_`, consoles, ROMs com filtros, download (multidisco vira zip com `.m3u`),
  BIOS, capas, saves/states/screenshots por usuário em
  `<data>/romm/...`, props do jogo.

Fora: HLS de áudio, cliente web embutido do Jellyfin, coleções do RomM,
podcasts/rádio/compartilhamento do Subsonic, controle remoto pelo websocket.

## Migração

`arkflix-server import` traz, uma vez, o que está nos três servidores de hoje.
Só lê os bancos de origem (SQLite em `mode=ro`, MariaDB numa transação
`READ ONLY`) e pode rodar quantas vezes quiser: não duplica nada, só junta.
Os itens casam pelo **caminho do arquivo**, então o acervo precisa estar
escaneado antes, e cada fonte vê a mídia num caminho diferente: `--map
ANTIGO=NOVO` (repete) troca o prefixo. `--dry-run` mostra o que mudaria; o
resumo lista os primeiros caminhos sem par (`--show N`) pra acertar o `--map`.

| Fonte | Traz | Fica de fora |
| --- | --- | --- |
| Navidrome (`navidrome.db`) | usuários com a senha de verdade (decifrada com `--key`), estrelas, vezes, nota e última vez de faixas/álbuns/artistas, fila, marcadores, playlists | playlists inteligentes, rádio, compartilhamentos, scrobbles |
| Jellyfin (`jellyfin.db`, 10.11+) | usuários (admin; desativado fica de fora), assistido, vezes, onde parou, última vez, favorito | preferências de exibição, aparelhos, coleções e playlists |
| RomM (MariaDB + `/romm/assets`) | usuários, saves/states/screenshots (copiados pra `<data>/romm/`), props do jogo (status, nota, backlog...), última vez, coleções | coleções inteligentes/virtuais, notas, memory cards, conquistas |

Senhas: o Navidrome guarda a senha cifrada, então o usuário já nasce com
ela e o token do Subsonic funciona na hora. O RomM guarda bcrypt e o
Jellyfin PBKDF2-SHA512: o hash é importado e o usuário entra com a senha
antiga; no **primeiro login** (Jellyfin, RomM ou Subsonic com `p=`) ela é
regravada e só a partir daí o token do Subsonic (`t=`/`s=`) funciona. Mesmo
nome em várias fontes vira um usuário só (sem diferenciar maiúsculas); quem
já tem senha fica com ela, a não ser com `--passwords=overwrite`. Por isso o
Navidrome vai primeiro. Usuário sem senha na fonte (OIDC no RomM, sem senha
no Jellyfin) precisa de `user passwd`.

Playlists e coleções ainda não têm destino no arkflix: por enquanto o
import só lista quais seriam trazidas.

Na VPS (stack em `/opt/media-stack`, com o serviço `arkflix-server` montando
`/var/lib/arkflix-server` em `/data` e `/var/media` em `/media`):

```sh
cd /opt/media-stack
ark() { docker compose run --rm "$@"; }

# 0. acervo escaneado (o import casa os itens pelo caminho)
ark arkflix-server scan

# 1. cópia dos bancos SQLite com o serviço parado (o WAL vai pro .db)
docker stop navidrome jellyfin
sudo install -d -o 1000 -g 1000 /var/lib/arkflix-server/import
sudo cp /var/lib/navidrome/navidrome.db /var/lib/jellyfin/config/data/jellyfin.db \
  /var/lib/arkflix-server/import/
sudo chown 1000:1000 /var/lib/arkflix-server/import/*.db
docker start navidrome jellyfin

# 2. Navidrome primeiro (senha de verdade). /music dele = /media/musicas aqui.
#    --key: o ND_PASSWORDENCRYPTIONKEY do Navidrome, se tiver; sem ele, omita.
ark arkflix-server import navidrome --db /data/import/navidrome.db \
  --map /music=/media/musicas --dry-run
ark arkflix-server import navidrome --db /data/import/navidrome.db \
  --map /music=/media/musicas

# 3. Jellyfin: ele vê a mídia em /media, igual o contêiner do arkflix
ark arkflix-server import jellyfin --db /data/import/jellyfin.db --dry-run
ark arkflix-server import jellyfin --db /data/import/jellyfin.db

# 4. RomM: credenciais do romm.env, saves de /var/lib/romm/assets
set -a; . /opt/media-stack/romm.env; set +a
ark -v /var/lib/romm/assets:/romm-assets:ro arkflix-server import romm \
  --dsn "$DB_USER:$DB_PASSWD@tcp(romm-db:3306)/${DB_NAME:-romm}" \
  --assets /romm-assets --map /romm/library=/media/jogos --dry-run
# (repete sem --dry-run)

# 5. limpeza
sudo rm -r /var/lib/arkflix-server/import
```

O `romm-db` precisa estar na mesma rede do contêiner (mesmo compose); se não
estiver, `--network <rede do romm-db>` no `docker compose run`. Rodando o
binário direto no host, os caminhos do arkflix são os do host:
`--map /media=/var/media` no Jellyfin, `--map /music=/var/media/musicas` no
Navidrome, `--map /romm/library=/var/media/jogos` e `--assets
/var/lib/romm/assets` no RomM.

## Desenvolvimento

```sh
make check-server   # gofmt, vet, testes com race + clientes do app contra o binário
```

Os testes usam `internal/testkit`: pastas com mídia pequena de verdade
(`testdata/`), banco em memória e o acervo de referência (`Seed`), que é o
contrato do scanner. Os repositórios do Jellyfin, Navidrome e RomM são
GPL/AGPL: serviram só de referência de comportamento, nenhum código foi copiado.

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

## Migração (ainda não feita)

Usuários, onde parou, estrelas, playlists e os saves que já estão no RomM não
vêm sozinhos. O próximo passo é um `arkflix-server import` lendo os bancos do
Jellyfin (SQLite), do Navidrome (SQLite) e do RomM (MariaDB + pasta de saves).

## Desenvolvimento

```sh
make check-server   # gofmt, vet, testes com race + clientes do app contra o binário
```

Os testes usam `internal/testkit`: pastas com mídia pequena de verdade
(`testdata/`), banco em memória e o acervo de referência (`Seed`), que é o
contrato do scanner. Os repositórios do Jellyfin, Navidrome e RomM são
GPL/AGPL: serviram só de referência de comportamento, nenhum código foi copiado.

# arkflix: instruções pra agentes

Leia o [README](README.md) antes. Resumo do que importa:

- Só o app é código deste repo. Os emuladores são os Flatpaks do Flathub
  (RetroArch, PPSSPP, PCSX2) e os cores vêm do buildbot do libretro
  (`make emulators`). Não volte a compilar emulador aqui.
- Build em `build/`, fora do git.
- Não mexa nas pastas dos Flatpaks (`~/.var/app/...`) nem no `~/ES-DE`: são os
  saves e a config reais do dono (o app só lê o `~/ES-DE`). Exceções: o
  `make emulators` e o `internal/emuinstall` (botão "Instalar emuladores")
  instalam emuladores e cores na pasta de cores do RetroArch (no `--fixture`,
  só nas pastas de teste), e o
  `internal/emucfg` ajusta só os atalhos de menu/saída do PPSSPP e do PCSX2
  (com backup). No RetroArch, nada vai no arquivo dele: tudo por `--appendconfig`,
  com `config_save_on_exit = "false"` (senão ele grava o acrescentado por cima).
- `make deps` usa sudo: peça pro dono rodar (`! make deps`).

## App (app/)

- Backend Go em `app/internal`, interface Svelte 5 + TypeScript em `app/web`.
  Métodos exportados de `core.Service` são a API (Wails no desktop, HTTP no web):
  não exporte método que não deva ser chamado pela interface.
- Fonte de jogos, metadados e saves entram por interface (`GameSource`,
  `MetaProvider`, `SaveStore`). O servidor RomM fica em `internal/romm` (cliente)
  e `internal/core/remote.go`; não acople nada remoto nos pacotes locais.
- Filmes e séries: cliente e proxy em `internal/jellyfin`, ligação com os perfis
  em `internal/core/watch.go` (conta por perfil, `jellyfin.json` 0600). O token
  nunca vai pra interface: imagem e vídeo passam pelo proxy em 127.0.0.1. Teste
  com `fixture.FakeJellyfin` (também em `/fake-jellyfin` no `--fixture --test`).
- Músicas: cliente Subsonic e proxy em `internal/subsonic`, ligação em
  `internal/core/music.go` (conta por perfil, `subsonic.json` 0600 só com
  t/s, nunca `p=`; cópias no PC em `musiclocal.go`). Teste com
  `fixture.FakeSubsonic` (`/fake-navidrome` no `--fixture --test`).
- arkflix-server: cliente em `internal/arkserver`, ligação em
  `internal/core/arkserver.go` (detecta pelo `/info`, usa o token do Jellyfin
  ou do RomM; contrato em `server/internal/api/admin/API.md`). Teste com
  `fixture.FakeArkServer` (`/fake-arkflix` no `--fixture --test`).
- Nunca guarde senha: o login do RomM vira um token de dispositivo. Teste com o
  RomM falso (`fixture.FakeRomM`, também em `/fake-romm` no `--fixture --test`;
  aceita Range e fica lento com `ARKFLIX_FAKE_ROMM_SLOW=<ms>`). A fila de
  downloads fica em `internal/core/downloads.go`.
- Mudou o app? `make check-app` (Go com race, svelte-check, Vitest, Playwright
  com axe). Tela nova ou mudada = teste de ponta a ponta + `axe(page, ...)` nela.
- Acessibilidade é requisito: contraste AA (paletas testadas no Go), nome
  acessível contendo o texto visível, foco sempre visível, tudo navegável por
  controle e teclado (`data-nav`).
- Dependência nova (Go ou npm): versão exata, publicada há pelo menos duas semanas.

## Servidor (server/)

- `arkflix-server` substitui Jellyfin, Navidrome e RomM falando a API dos três
  (leia `server/README.md`). Só lê as pastas de mídia; escreve só em `--data`.
- Núcleo em `internal/{store,auth,library,media}`; cada API no seu pacote
  (`internal/api/{jellyfin,subsonic,romm}`), com tabelas próprias via
  `store.Register`. Nunca copie código do Jellyfin/Navidrome/RomM (GPL/AGPL).
- Mudou o servidor? `make check-server` (inclui os clientes do app contra o
  binário). Commits de `server/` não entram no changelog do app.

## Visual

Segue a skill `noemi-ui` (Noemi): neobrutalismo sóbrio com tempero de desenho
animado dos anos 30. Decisões já tomadas:

- **Tema escuro de cinema antigo, fixo** (tokens em `app.css`, `--n-*`).
  Três acentos e um sentido cada: mostarda = ação/destaque, verde-azulado =
  ok/info/foco, vermelho = erro. **Sem cor por console** (`applyPalette` só
  limpa); o console aparece como etiqueta/nome, nunca pintando a tela.
- Fontes locais (`@fontsource`): Limelight (títulos), Bevan (rótulos e
  botões), Alegreya (texto), Special Elite (meta, etiquetas), Courier Prime
  (código). Nada de fonte pixel, neon ou synthwave.
- Sóbrio por padrão (borda reta 3px, `--radii-sm`, sombra dura `--hard`); no
  máximo 2 tipos de componente de desenho (`.sketched`, `.stamp`...) por tela.
- **CRT só em momento** (`lib/tube.svelte.ts`): abrir o app, começar filme,
  fechar jogo (~1,6 s, depois some). **Glitch só por evento**: erro =
  estática, trocar perfil = desmagnetizar. Nada de ambiente.
- **Aberração cromática** (filtros `#chroma`/`#chroma-strong` em
  `components/Sketch.svelte`): leve e parada nos `h1`, forte no momento de
  CRT; nunca no texto corrido. **Textura de filme** (`components/Film.svelte`):
  grão pintado uma vez + vinheta, parada. Alto contraste desliga os dois.
- **Nada anima em loop** (WebKit repinta e gasta CPU): animação em degraus
  (`steps`), curta, ligada a um evento. Troca de tela = swap; recomeço = íris.
- Sons de navegação ficam. Alto contraste troca os tokens; "reduzir
  animações" desliga tudo.

## Git: commit + push sempre

Toda iteração que mudar arquivo termina com commit e push. Sem perguntar.

- Antes de responder: `git status`, `git add` só do que você mudou, commit,
  `git push` no `master`. Nada de `git add -A` às cegas.
- Conventional Commits em português, imperativo, 72 colunas por linha, sem emoji,
  sem Co-Authored-By. Um commit por mudança lógica.
- Push falhou? Diga ao dono. Nunca `--force`.
- Versão e release são automáticos (CI, pelo tipo do commit): escolha o tipo
  certo, ele decide a versão. `feat`/`fix` geram release; `docs`/`chore` não.
- `make release-binaries` publica binários no GitHub: só quando o dono pedir.

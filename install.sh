#!/usr/bin/env bash
# Instala o arkflix: curl -fsSL <raw>/install.sh | bash
#
# Clona (ou atualiza) o repo, instala as dependências (apt, pede sudo), os
# emuladores (Flatpaks do Flathub e os cores do RetroArch), baixa o app do
# último release (sem release com binários: compila, ~1 min) e cria o comando
# `arkflix`. Rodar de novo = atualizar.
#
# Variáveis: ARKFLIX_DIR (padrão ~/.local/share/arkflix),
#            ARKFLIX_BUILD=1 (compila mesmo tendo release),
#            ARKFLIX_NO_DEPS=1 (pula o apt),
#            ARKFLIX_LAUNCHER (comando, padrão ~/.local/bin/arkflix).
set -euo pipefail

repo_url=https://github.com/stanleygomes/arkflix.git
dir=${ARKFLIX_DIR:-$HOME/.local/share/arkflix}
launcher=${ARKFLIX_LAUNCHER:-$HOME/.local/bin/arkflix}
mark="# gerado pelo install.sh do arkflix"

die() { echo "❌ $*" >&2; exit 1; }
step() { printf '\n\033[1m▶ %s\033[0m\n' "$*"; }

[[ $(uname -sm) == "Linux x86_64" ]] || die "só Linux x86_64"
command -v apt-get >/dev/null || die "precisa de Ubuntu/Debian (apt)"

if [[ -z ${ARKFLIX_NO_DEPS:-} ]]; then
  step "pacotes básicos (sudo)"
  sudo apt-get update -qq
  sudo apt-get install -y -qq git make curl ca-certificates python3 >/dev/null
fi

# antes o app se chamava arkflix-games: aproveita o clone e tira o comando velho
old=$HOME/.local/share/arkflix-games
if [[ -z ${ARKFLIX_DIR:-} && ! -d $dir/.git && -d $old/.git ]]; then
  step "renomeando $old para $dir"
  # o app novo já pode ter aberto: o WebKit cria $dir com os dados dele
  if [[ -d $dir ]]; then
    cp -an "$dir/." "$old/"
    rm -rf "$dir"
  fi
  mv "$old" "$dir"
  git -C "$dir" remote set-url origin "$repo_url"
fi
if [[ -f $HOME/.local/bin/arkflix-games ]] && grep -qF "gerado pelo install.sh do arkflix" "$HOME/.local/bin/arkflix-games"; then
  rm -f "$HOME/.local/bin/arkflix-games"
fi

step "código em $dir"
if [[ -d $dir/.git ]]; then
  git -C "$dir" pull -q --ff-only
else
  # init + fetch em vez de clone: funciona com a pasta já existindo (o WebKit
  # do app guarda os dados dele aqui)
  mkdir -p "$dir"
  git -C "$dir" init -q -b master
  git -C "$dir" remote add origin "$repo_url" 2>/dev/null || git -C "$dir" remote set-url origin "$repo_url"
  git -C "$dir" fetch -q --depth 1 origin master:refs/remotes/origin/master
  git -C "$dir" checkout -q -f -B master origin/master
  git -C "$dir" branch -q --set-upstream-to=origin/master master
fi
cd "$dir"

if [[ -z ${ARKFLIX_NO_DEPS:-} ]]; then
  step "dependências (sudo)"
  make deps
fi

step "emuladores (Flatpak)"
make emulators

if [[ -z ${ARKFLIX_BUILD:-} ]] && make release-install; then
  :
else
  step "sem binários prontos: compilando o app"
  make app
fi

step "comando $(basename "$launcher")"
# nunca escreve por cima de um comando que não é deste app (nem através de
# link: o cat seguiria o link e trocaria o arquivo de outro programa)
if [[ -L $launcher ]] || { [[ -e $launcher ]] && ! grep -qF "$mark" "$launcher"; }; then
  echo "⚠️  $launcher já existe e não é deste app: deixei como está."
  echo "   Abra pelo menu de aplicativos ou rode de novo com ARKFLIX_LAUNCHER=~/.local/bin/outro-nome"
else
  mkdir -p "$(dirname "$launcher")"
  cat >"$launcher" <<LAUNCHER
#!/bin/sh
$mark
exec make --no-print-directory -C "$dir" run "\$@"
LAUNCHER
  chmod +x "$launcher"
fi

echo
echo "✅ instalado. Abrir: $(basename "$launcher") (ou pelo menu de aplicativos)"
case :$PATH: in *":$HOME/.local/bin:"*) ;; *) echo "   (~/.local/bin fora do PATH: abra um terminal novo)" ;; esac

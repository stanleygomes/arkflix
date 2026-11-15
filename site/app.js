// Detecta o sistema de quem visita e destaca o download dele. Os links são
// fixos (.../releases/latest/download/<arquivo sem versão>): o GitHub sempre
// entrega o da última release. A API só mostra o número da versão (se cair ou
// passar do limite, nada quebra). Sem innerHTML: só textContent e append.
const REPO = 'stanleygomes/arkflix'
const LABELS = { windows: 'Baixar pra Windows', macos: 'Baixar pra macOS', linux: 'Baixar pra Linux' }
const MAIN_ASSET = { windows: 'windows-x86_64-setup.exe', macos: 'macos-universal.dmg', linux: 'linux-x86_64.tar.gz' }
const DOWNLOAD = `https://github.com/${REPO}/releases/latest/download/arkflix-`
const reduceMotion = matchMedia('(prefers-reduced-motion: reduce)').matches

function detectOS() {
  const ua = `${navigator.userAgentData?.platform ?? ''} ${navigator.platform ?? ''} ${navigator.userAgent}`.toLowerCase()
  if (/android|iphone|ipad/.test(ua)) return null // celular: mostra as três opções
  if (ua.includes('win')) return 'windows'
  if (ua.includes('mac')) return 'macos'
  if (ua.includes('linux') || ua.includes('x11')) return 'linux'
  return null
}

async function latest() {
  const res = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`, { headers: { Accept: 'application/vnd.github+json' } })
  if (!res.ok) throw new Error(`GitHub ${res.status}`)
  return res.json()
}

function showVersion(release) {
  document.getElementById('site-version').textContent = `· ${release.tag_name}`
  const a = document.createElement('a')
  a.href = release.html_url
  a.textContent = `${release.tag_name} · novidades`
  document.getElementById('release-line').replaceChildren('Última versão: ', a)
}

// 🎬 carimbo de tinta no cartão do sistema de quem visita: bate uma vez
function stamp(card) {
  const mark = document.createElement('span')
  mark.className = reduceMotion ? 'stamp' : 'stamp thunk'
  mark.textContent = 'Seu sistema'
  card.append(mark)
}

function highlight(os) {
  if (!os) return
  const card = document.querySelector(`[data-os="${os}"]`)
  if (card) {
    card.classList.add('mine')
    stamp(card)
  }
  const cta = document.getElementById('cta-main')
  document.getElementById('cta-label').textContent = LABELS[os]
  if (os === 'linux') {
    cta.href = '#bilheteria' // no Linux o melhor é o comando, que traz os emuladores
    document.getElementById('cta-note').textContent = 'Um comando instala o app e os emuladores · também pra Windows e macOS'
    return
  }
  cta.href = DOWNLOAD + MAIN_ASSET[os]
  document.getElementById('cta-note').textContent = 'Grátis e de código aberto · outros sistemas na bilheteria, mais abaixo'
}

document.getElementById('copy-cmd')?.addEventListener('click', async (ev) => {
  const btn = ev.currentTarget
  const text = document.getElementById('linux-cmd').textContent
  try {
    await navigator.clipboard.writeText(text)
    btn.textContent = 'Copiado'
  } catch {
    btn.textContent = 'Selecione e copie'
  }
  setTimeout(() => { btn.textContent = 'Copiar' }, 2000)
})

// celular: o app é de computador; manda o link pra lá (compartilhar ou copiar)
document.getElementById('share-link')?.addEventListener('click', async (ev) => {
  const btn = ev.currentTarget
  const url = location.origin + '/'
  try {
    if (navigator.share) await navigator.share({ title: 'arkflix', text: 'Baixar no computador:', url })
    else {
      await navigator.clipboard.writeText(url)
      btn.textContent = 'Link copiado'
    }
  } catch {
    btn.textContent = url
  }
})

// textura de filme: grão parado, pintado uma vez num canvas pequeno esticado
function paintGrain() {
  const canvas = document.createElement('canvas')
  canvas.className = 'grain'
  canvas.width = 320
  canvas.height = 200
  canvas.setAttribute('aria-hidden', 'true')
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  let seed = 7 // semente fixa: o grão é sempre o mesmo
  const rand = () => ((seed = (seed * 16807) % 2147483647) / 2147483647)
  for (let i = 0; i < 2600; i++) {
    ctx.fillStyle = `rgb(241 228 198 / ${(0.04 + rand() * 0.08).toFixed(3)})`
    ctx.fillRect(rand() * 320, rand() * 200, 1, 1)
  }
  document.body.append(canvas)
}

highlight(detectOS())
paintGrain()
latest().then(showVersion).catch(() => {})

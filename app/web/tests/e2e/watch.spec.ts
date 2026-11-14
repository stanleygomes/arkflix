import { axe, expect, open, test, type App } from './fixtures'
import type { Page } from '@playwright/test'

/** Vincula o Jellyfin falso ao perfil em uso, a partir da aba Filmes e séries. */
async function link(page: Page, app: App) {
  await page.locator('[data-section="watch"]').click()
  await expect(page.getByRole('heading', { name: 'Filmes e séries' })).toBeVisible()
  await page.getByRole('button', { name: 'Vincular conta do Jellyfin' }).click()
  const accounts = page.locator('fieldset.accounts')
  await expect(accounts).toBeVisible()
  await accounts.getByRole('button', { name: /^Endereço/ }).click()
  await page.getByRole('dialog', { name: 'Endereço do Jellyfin' }).getByLabel('Endereço do Jellyfin').fill(`${app.url}/fake-jellyfin`)
  await page.keyboard.press('Enter')
  await accounts.getByRole('button', { name: /^Usuário/ }).click()
  await page.getByRole('dialog', { name: 'Usuário do Jellyfin' }).getByLabel('Usuário do Jellyfin').fill('stanley')
  await page.keyboard.press('Enter')
  await accounts.getByRole('button', { name: 'Vincular' }).click()
  return accounts
}

test('filmes e séries: vincula a conta, navega, toca e lembra onde parou', async ({ page, app }) => {
  await open(page, app)
  await page.locator('[data-section="watch"]').click()
  await expect(page.getByRole('button', { name: 'Vincular conta do Jellyfin' })).toBeVisible()
  await axe(page, 'filmes sem conta')
  await page.locator('[data-section="games"]').click()

  const accounts = await link(page, app)
  const pass = page.getByRole('dialog', { name: 'Senha do Jellyfin' }).getByLabel('Senha do Jellyfin')
  await expect(pass).toHaveAttribute('type', 'password')
  await pass.fill('errada')
  await page.keyboard.press('Enter')
  await expect(page.getByRole('alert')).toContainText('senha')
  await pass.fill('senha123')
  await page.keyboard.press('Enter')
  await expect(accounts).toContainText('Conectado a Jellyfin de Teste como stanley')
  const saved = app.read(`${app.data}/jellyfin.json`)
  expect(saved).toContain('tok-stanley')
  expect(saved).not.toContain('senha123')
  await axe(page, 'contas do perfil')

  // volta pra aba: destaque no que parou no meio, fileiras do servidor
  await page.keyboard.press('Escape')
  await expect(page.getByRole('heading', { name: 'O Grande Teste' })).toBeVisible()
  await expect(page.locator('section.shelf', { hasText: 'Continuar assistindo' }).locator('[data-item]')).toHaveCount(1)
  await expect(page.locator('section.shelf', { hasText: 'Próximos episódios' }).locator('[data-item="e2"]')).toHaveAttribute('aria-label', 'Série de Teste T1:E2 · O Segundo')
  await expect(page.locator('.hero .meta')).toContainText('faltam 1 h 30 min')
  await axe(page, 'início de filmes e séries')

  // aba Filmes: grade com ordem
  await page.locator('[data-watch-tab="movies"]').click()
  await expect(page.locator('.grid [data-item]')).toHaveCount(3)
  await page.getByRole('button', { name: 'A-Z' }).click()
  await expect(page.locator('.grid [data-item]').first()).toHaveAttribute('aria-label', /^Curta Pixel/)
  await axe(page, 'lista de filmes')

  // série: temporadas e episódios, próximo a assistir
  await page.locator('[data-watch-tab="series"]').click()
  await page.locator('.grid [data-item="s1"]').click()
  await expect(page.getByRole('button', { name: 'Assistir T1:E2' })).toBeVisible()
  await expect(page.locator('.list [data-episode]')).toHaveCount(3)
  await page.getByRole('button', { name: 'Temporada 2', exact: true }).click()
  await expect(page.locator('.list [data-episode]')).toHaveCount(1)
  await expect(page.locator('.list')).toContainText('Recomeço')
  const fav = page.getByRole('button', { name: 'Minha lista' })
  await expect(fav).toHaveAttribute('aria-pressed', 'false')
  await fav.click()
  await expect(fav).toHaveAttribute('aria-pressed', 'true')
  await axe(page, 'página da série')
  await page.keyboard.press('Escape')

  // filme novo: toca do começo pelo proxy, o servidor guarda onde parou
  await page.locator('[data-watch-tab="movies"]').click()
  await page.locator('.grid [data-item="m3"]').click()
  await expect(page.getByRole('button', { name: 'Do início' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Assistir', exact: true }).click()
  const player = page.locator('main.player')
  await expect(player).toBeVisible()
  await expect(player.getByRole('heading', { name: 'Curta Pixel' })).toBeVisible()
  const src = await page.locator('main.player video').evaluate((v: HTMLVideoElement) => v.currentSrc)
  expect(src).toMatch(/^http:\/\/127\.0\.0\.1:\d+\/[0-9a-f]{32}\/Videos\/m3\/stream\?/)
  expect(src).not.toContain('tok-stanley')
  await page.locator('main.player video').evaluate((v: HTMLVideoElement) => v.pause())
  await expect(player.getByRole('button', { name: 'Tocar' })).toBeVisible()
  await axe(page, 'player')

  // áudio e legendas
  await player.getByRole('button', { name: 'Áudio e legendas' }).click()
  const tracks = page.getByRole('dialog', { name: 'Áudio e legendas' })
  await expect(tracks.getByRole('radio', { name: 'Português - Stereo' })).toHaveAttribute('aria-checked', 'true')
  await tracks.getByRole('radio', { name: 'Português', exact: true }).click()
  await expect(tracks.getByRole('radio', { name: 'Português', exact: true })).toHaveAttribute('aria-checked', 'true')
  await axe(page, 'áudio e legendas')
  await tracks.getByRole('button', { name: 'Fechar' }).click()

  // toca até o fim (clipe de 2 s): volta sozinho pra página, que agora continua
  await player.getByRole('button', { name: 'Tocar' }).click()
  await expect(player).toHaveCount(0, { timeout: 10_000 })
  await expect(page.getByRole('button', { name: 'Continuar' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Do início' })).toBeVisible()

  // busca acha filme e série junto com os jogos
  await page.keyboard.press('/')
  await page.getByLabel('Buscar jogo, console ou configuração').fill('pixel')
  await expect(page.locator('.videos [data-item="m3"]')).toBeVisible()
  await axe(page, 'busca com filmes')
})

test('filmes e séries: player no controle (pausa, pula e sai)', async ({ page, app }) => {
  await open(page, app)
  await link(page, app)
  await page.getByRole('dialog', { name: 'Senha do Jellyfin' }).getByLabel('Senha do Jellyfin').fill('senha123')
  await page.keyboard.press('Enter')
  await expect(page.locator('fieldset.accounts')).toContainText('Conectado')
  await page.keyboard.press('Escape')

  // episódio da fileira abre direto no player
  await page.locator('section.shelf', { hasText: 'Próximos episódios' }).locator('[data-item="e2"]').click()
  const player = page.locator('main.player')
  await expect(player).toBeVisible()
  await expect(player.getByRole('heading', { name: 'Série de Teste' })).toBeVisible()
  await expect(player).toContainText('T1:E2 · O Segundo')
  await page.locator('main.player video').evaluate((v: HTMLVideoElement) => v.pause())
  await expect(player.getByRole('button', { name: 'Tocar' })).toBeFocused()

  // Start pausa/toca; B sai
  await app.pad('start')
  await expect(player.getByRole('button', { name: 'Pausar' })).toBeVisible()
  await app.pad('start')
  await expect(player.getByRole('button', { name: 'Tocar' })).toBeVisible()
  await expect(player.getByRole('button', { name: 'Próximo episódio' })).toBeVisible()
  await app.pad('east')
  await expect(player).toHaveCount(0)
  // parou no meio: o episódio entra em Continuar assistindo
  await expect(page.locator('section.shelf', { hasText: 'Continuar assistindo' }).locator('[data-item="e2"]')).toBeVisible()
})

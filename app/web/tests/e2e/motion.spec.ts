import type { Page } from '@playwright/test'
import { expect, open, switchProfile, test } from './fixtures'

test.use({ reducedMotion: 'no-preference' })

// animações em andamento agora (Web Animations e CSS)
const running = (page: Page) => page.evaluate(() => document.getAnimations().filter((a) => a.playState === 'running').length)

test('animações: telas, abas, capa em foco, título, estrela e PLAYER 1 START', async ({ page, app }) => {
  await open(page, app)
  const card = page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first()
  await card.focus()
  // capa acende e respira; título digitado
  await expect.poll(() => card.evaluate((el) => getComputedStyle(el).animationName)).toContain('breathe')
  await expect(page.locator('#hero-title')).toHaveClass(/typing/)

  // estrela em pixels ao favoritar
  await page.keyboard.press('f')
  await expect(page.locator('.burst-pixel')).toHaveCount(8)
  await expect(page.locator('.burst-pixel')).toHaveCount(0, { timeout: 2000 })

  // aba: a luz desliza até o console
  await page.keyboard.press('e')
  const light = page.locator('.tab-light')
  await expect(light).toHaveCount(1)
  const current = page.locator('nav.tabs [aria-current="page"]')
  await expect.poll(async () => {
    const [l, c] = await Promise.all([light.boundingBox(), current.boundingBox()])
    return l && c ? Math.abs(l.x - c.x) < 2 && Math.abs(l.width - c.width) < 2 : false
  }).toBe(true)

  // troca de tela curta (swap); durante a troca, a tela que sai fica inerte
  const seen = page.evaluate(
    () =>
      new Promise<number>((resolve) => {
        let max = 0
        const t0 = performance.now()
        const tick = () => {
          max = Math.max(max, document.getAnimations().length)
          if (performance.now() - t0 < 400) requestAnimationFrame(tick)
          else resolve(max)
        }
        tick()
      }),
  )
  await page.keyboard.press('m')
  expect(await seen).toBeGreaterThan(0)
  await expect(page.locator('.layer[inert]')).toHaveCount(0, { timeout: 2000 })
  await expect(page.locator('.layer')).toHaveCount(1)
  await expect(page.locator('main.you :focus')).toHaveCount(1) // foco caiu na tela nova
  await page.keyboard.press('Escape')
  await expect(page.locator('.layer')).toHaveCount(1, { timeout: 2000 })

  // jogar: PLAYER 1 START sobre a capa
  await page.keyboard.press('q') // volta pro Início
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
  await page.getByRole('button', { name: 'Jogar do início' }).click()
  await expect(page.getByText('PLAYER 1 START')).toBeVisible()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
})

test('trocar de perfil abre a tela nova numa íris', async ({ page, app }) => {
  await open(page, app)
  await switchProfile(page)
  await page.getByRole('button', { name: /^Jogador 1/ }).click()
  await expect(page.locator('.iris')).toHaveCount(1)
  await expect(page.locator('.iris')).toHaveCount(0, { timeout: 5000 })
  // depois, nada fica animando (só o que for infinito por engano aparece aqui)
  const left = await page.evaluate(() => document.getAnimations().filter((a) => a.playState === 'running').map((a) => (a as CSSAnimation).animationName ?? (a.effect as KeyframeEffect)?.target?.className))
  expect(left).toEqual([])
})

test.describe('com reduzir animações', () => {
  test.use({ reducedMotion: 'reduce' })
  test('nada se mexe: sem estrela, sem íris, troca de tela seca', async ({ page, app }) => {
    await open(page, app)
    await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().focus()
    await page.keyboard.press('f')
    await page.keyboard.press('m')
    await expect(page.locator('.layer')).toHaveCount(1)
    expect(await page.locator('.burst-pixel').count()).toBe(0)
    expect(await page.locator('.layer[inert]').count()).toBe(0)
  })
})

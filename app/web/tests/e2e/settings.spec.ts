import { axe, expect, open, test, settings, pickItem } from './fixtures'

test('acessibilidade: fonte, alto contraste e animações aplicam na hora e persistem', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await pickItem(page, 'Acessibilidade e som')
  await axe(page, 'configurações: acessibilidade')

  await page.getByRole('button', { name: '150%' }).click()
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).fontSize)).toBe('25.5px') // 17px × 1,5

  await page.getByRole('switch', { name: 'Alto contraste' }).click()
  await expect(page.locator('body')).toHaveClass(/high-contrast/)
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.body).backgroundColor)).toBe('rgb(0, 0, 0)')
  await axe(page, 'alto contraste')

  await page.getByRole('switch', { name: 'Reduzir animações' }).click()
  await expect(page.locator('body')).toHaveClass(/reduce-motion/)
  await page.getByRole('button', { name: 'Diminuir volume da música' }).click()
  await expect(page.getByRole('meter', { name: 'Volume da música' })).toHaveAttribute('aria-valuenow', '15')

  await expect.poll(() => JSON.parse(app.read(`${app.data}/settings.json`)).a11y.highContrast).toBe(true)
  await page.reload()
  await expect(page.locator('body')).toHaveClass(/high-contrast/)
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).fontSize)).toBe('25.5px') // 17px × 1,5
})

test('tipografia da noemi-ui: título em Limelight, texto em Alegreya', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  const h1 = await page.locator('h1').first().evaluate((el) => getComputedStyle(el).fontFamily)
  expect(h1).toContain('Limelight')
  const body = await page.locator('body').evaluate((el) => getComputedStyle(el).fontFamily)
  expect(body).toContain('Alegreya')
  await expect.poll(() => page.evaluate(() => document.fonts.check('16px Limelight'))).toBe(true)
  // aberração cromática leve nos títulos e textura de filme parada no app
  expect(await page.locator('h1').first().evaluate((el) => getComputedStyle(el).textShadow)).toContain('rgba(224, 87, 74')
  expect(await page.locator('body').evaluate((el) => getComputedStyle(el).filter)).toBe('none')
  await expect(page.locator('.film canvas')).toHaveCount(1)
  expect(await page.evaluate(() => document.getAnimations().filter((a) => a.playState === 'running').length)).toBe(0)
  // alto contraste tira os dois
  await pickItem(page, 'Acessibilidade e som')
  await page.getByRole('switch', { name: 'Alto contraste' }).click()
  await expect(page.locator('.film')).toBeHidden()
  expect(await page.locator('h1').first().evaluate((el) => getComputedStyle(el).textShadow)).toBe('none')
})

test('biblioteca: adicionar pasta e salvar procura os jogos de novo', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await expect(page.getByRole('heading', { name: 'Biblioteca', level: 1 })).toBeVisible()
  await axe(page, 'configurações: biblioteca')
  const save = page.getByRole('button', { name: 'Salvar' })
  await expect(save).toBeDisabled()
  await page.getByRole('button', { name: 'Remover pasta' }).last().click()
  await save.click()
  await expect(page.locator('.toasts')).toContainText('6 jogos na biblioteca')
  const saved = JSON.parse(app.read(`${app.data}/settings.json`))
  expect(saved.libraryDirs).toHaveLength(1)
})

test('sobre: versão e atalhos', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await page.keyboard.press('Escape') // do conteúdo volta pra lista
  await pickItem(page, 'Sobre')
  await expect(page.getByRole('heading', { name: 'Sobre' })).toBeVisible()
  await expect(page.locator('.facts').first()).toContainText('Navegador')
  await expect(page.getByRole('definition').filter({ hasText: '10-registra.sh' })).toBeVisible()
  await axe(page, 'configurações: sobre')
})

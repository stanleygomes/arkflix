import type { Page } from '@playwright/test'
import { axe, expect, gameOption, open, test, you, type App } from './fixtures'

const call = async (page: Page, app: App, m: string, args: unknown[] = []) => {
  const res = await page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
  const body = await res.json()
  expect(res.ok(), `${m}: ${body.error}`).toBeTruthy()
  return body.result
}

/** Liga o perfil (Jellyfin) e o app (RomM) no arkflix-server falso. */
async function link(page: Page, app: App, profile = 'main') {
  const base = `${app.url}/fake-arkflix`
  await call(page, app, 'ConnectWatch', [profile, base, 'stanley', 'senha123'])
  if (profile === 'main') await call(page, app, 'ConnectRemote', [base, 'stanley', 'senha123'])
  // o catálogo do RomM chega em segundo plano: espera os jogos do servidor
  await expect.poll(async () => (await call(page, app, 'RemoteStatus')).games).toBe(3)
}

/** Abre um item da tela Você pelo controle (foco + Enter). */
async function youItem(page: Page, item: string, tab?: string) {
  await you(page)
  await page.locator('main.you [data-item]').filter({ has: page.getByText(item, { exact: true }) }).focus()
  await page.keyboard.press('Enter')
  if (tab) {
    const btn = page.locator('main.you .subtabs').getByRole('button', { name: tab, exact: true })
    await btn.focus()
    await page.keyboard.press('Enter')
  }
}

test('servidor arkflix: estado, scan, usuários e envio de arquivos', async ({ page, app }) => {
  await page.goto(app.url)
  await link(page, app)
  await open(page, app)

  await youItem(page, 'Servidor arkflix')
  await expect(page.getByRole('heading', { name: 'Estado', level: 1 })).toBeVisible()
  const libs = page.locator('main.you table')
  await expect(libs.locator('tbody tr')).toHaveCount(4)
  await expect(libs.locator('tbody tr', { hasText: 'Músicas' })).toContainText('Fora do ar')
  await expect(page.locator('main.you .status')).toContainText('Gravável')
  await expect(page.getByText('Último scan')).toContainText('1500 itens')
  await axe(page, 'estado do servidor arkflix')

  // escanear: a tarefa acompanha e o painel recarrega no fim
  await page.getByRole('button', { name: 'Escanear agora' }).click()
  await expect(page.getByText('Último scan')).toContainText('1501 itens', { timeout: 10_000 })

  // usuários: criar, tornar admin, apagar (telegrama)
  await page.locator('main.you .subtabs').getByRole('button', { name: 'Usuários', exact: true }).click()
  const rows = page.locator('main.you tbody tr')
  await expect(rows).toHaveCount(2)
  await expect(rows.first()).toContainText('Você')
  await page.getByRole('button', { name: /^Nome/ }).click()
  await page.getByRole('dialog', { name: 'Nome do usuário' }).getByLabel('Nome do usuário').fill('bia')
  await page.keyboard.press('Enter')
  await page.getByRole('button', { name: /^Senha/ }).click()
  const pass = page.getByRole('dialog', { name: 'Senha do usuário' }).getByLabel('Senha do usuário')
  await expect(pass).toHaveAttribute('type', 'password')
  await pass.fill('segredo1')
  await page.keyboard.press('Enter')
  await page.getByRole('switch', { name: /Administrador/ }).click()
  await page.getByRole('button', { name: 'Criar usuário' }).click()
  await expect(rows.filter({ hasText: 'bia' })).toContainText('Admin')
  await axe(page, 'usuários do servidor')
  await page.getByRole('button', { name: 'Tirar admin de bia' }).click()
  await expect(rows.filter({ hasText: 'bia' })).not.toContainText('Admin')
  await page.getByRole('button', { name: 'Apagar ana' }).click()
  const dialog = page.getByRole('alertdialog')
  await expect(dialog).toContainText('Telegrama')
  await axe(page, 'apagar usuário')
  await dialog.getByRole('button', { name: 'Apagar usuário' }).click()
  await expect(rows).toHaveCount(2)
  await expect(rows.filter({ hasText: 'ana' })).toHaveCount(0)

  // enviar: ROM pede o console; o progresso aparece aqui e na Central
  app.write('inbox/Chrono Trigger.sfc', 'SNES-CT')
  await page.locator('main.you .subtabs').getByRole('button', { name: 'Enviar arquivos', exact: true }).click()
  await page.getByRole('radio', { name: 'ROM' }).click()
  await page.getByLabel('Caminho de um arquivo').fill(`${app.root}/inbox/Chrono Trigger.sfc`)
  await page.getByRole('button', { name: 'Adicionar' }).click()
  const send = page.getByRole('button', { name: 'Enviar pro servidor' })
  await expect(send).toBeDisabled()
  await page.getByRole('radio', { name: 'Super Nintendo' }).click()
  await axe(page, 'enviar arquivos')
  await send.click()
  await expect(page.locator('main.you .progress')).toContainText('Envio terminado: 1 arquivo no servidor')
})

test('servidor arkflix: listas em filmes e jogos, esconder do servidor', async ({ page, app }) => {
  await page.goto(app.url)
  await link(page, app)
  await open(page, app)

  // jogos: a lista do servidor vira fileira do Início (só o que está aqui)
  const shelf = page.locator('section.shelf', { hasText: 'Sessão de sábado' })
  await expect(shelf.locator('[data-game]')).toHaveCount(1)
  await expect(shelf.getByText('(lista do servidor)')).toBeAttached()

  // filmes e séries: aba Listas com uma fileira por lista
  await page.locator('[data-section="watch"]').click()
  await page.locator('[data-watch-tab="lists"]').click()
  const row = page.locator('section.shelf', { hasText: 'Sessão de sábado' })
  await expect(row.locator('[data-item]')).toHaveCount(2)
  await expect(row.locator('[data-item="m1"] img')).toHaveAttribute('src', /\/arkflix\/api\/items\/m1\/image/)
  await axe(page, 'aba listas')

  // abrir a lista: grade, jogos e tirar item (botão West)
  await row.getByRole('button', { name: 'Abrir lista Sessão de sábado' }).click()
  await expect(page.getByRole('heading', { name: 'Sessão de sábado', level: 1 })).toBeVisible()
  const grid = page.locator('.grid [data-item]')
  await expect(grid).toHaveCount(2)
  await expect(page.getByRole('button', { name: 'Ver os jogos (2)' })).toBeVisible()
  await axe(page, 'lista do servidor')
  await grid.first().focus()
  await page.keyboard.press('i')
  await expect(grid).toHaveCount(1)
  await expect(page.getByText('saiu da lista')).toBeVisible()

  // jogos da lista na Biblioteca (o remoto também)
  await page.getByRole('button', { name: 'Ver os jogos (2)' }).click()
  await expect(page.locator('.grid [data-game]')).toHaveCount(2)
  await axe(page, 'jogos da lista do servidor')
  await page.keyboard.press('Escape')
  await page.keyboard.press('Escape')

  // página do filme: adicionar à lista (e criar uma) e esconder do servidor
  await page.locator('[data-watch-tab="movies"]').click()
  await page.locator('.grid [data-item="m3"]').click()
  await page.getByRole('button', { name: 'Adicionar à lista' }).click()
  const picker = page.getByRole('dialog', { name: 'Listas do servidor' })
  const check = picker.getByRole('checkbox', { name: /Sessão de sábado/ })
  await expect(check).toHaveAttribute('aria-checked', 'false')
  await check.click()
  await expect(check).toHaveAttribute('aria-checked', 'true')
  await picker.getByRole('button', { name: 'Nova lista' }).click()
  await page.getByRole('dialog', { name: 'Nova lista do servidor' }).getByLabel('Nova lista do servidor').fill('Pra Nina')
  await page.keyboard.press('Enter')
  await expect(picker.getByRole('checkbox', { name: /Pra Nina/ })).toHaveAttribute('aria-checked', 'true')
  await axe(page, 'listas do servidor no filme')
  await picker.getByRole('button', { name: 'Pronto' }).click()
  await page.getByRole('button', { name: 'Esconder do servidor' }).click()
  await expect(page.getByRole('alertdialog')).toContainText('Curta Pixel')
  await page.getByRole('alertdialog').getByRole('button', { name: 'Esconder do servidor' }).click()
  await expect(page.getByText('"Curta Pixel" escondido do servidor')).toBeVisible()
  await expect(page.locator('.grid [data-item="m3"]')).toBeVisible() // voltou pra grade

  // jogo do servidor: listas de lá e esconder no menu Opções
  await page.locator('[data-section="games"]').click()
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
  await gameOption(page, 'Listas do servidor')
  const gp = page.getByRole('dialog', { name: 'Listas do servidor' })
  await expect(gp.getByRole('checkbox', { name: /Sessão de sábado/ })).toHaveAttribute('aria-checked', 'true')
  await gp.getByRole('button', { name: 'Pronto' }).click()
  await page.getByRole('button', { name: 'Opções', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Opções' }).getByRole('button', { name: 'Esconder do servidor' })).toBeVisible()
})

test('servidor arkflix: perfil infantil só vê as listas', async ({ page, app }) => {
  await page.goto(app.url)
  const nina = await call(page, app, 'CreateProfile', ['Nina', 'mushroom', '#d8579a'])
  await call(page, app, 'SetProfileKids', [nina.id, true])
  await link(page, app)
  await link(page, app, nina.id)
  await page.goto(app.url)
  await page.getByRole('button', { name: /^Nina/ }).click()
  await expect(page.getByRole('button', { name: /^Você: Nina/ })).toBeVisible()

  await page.locator('[data-section="watch"]').click()
  await page.locator('[data-watch-tab="lists"]').click()
  await page.getByRole('button', { name: 'Abrir lista Sessão de sábado' }).click()
  await expect(page.locator('.grid [data-item]')).toHaveCount(2)
  await expect(page.getByRole('button', { name: 'Renomear' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Apagar lista' })).toHaveCount(0)
  await axe(page, 'lista do servidor no infantil')
  await page.locator('.grid [data-item="m1"]').click()
  await expect(page.getByRole('heading', { name: 'O Grande Teste' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Adicionar à lista' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Esconder do servidor' })).toHaveCount(0)

  await page.getByRole('button', { name: /^Você: Nina/ }).click()
  await expect(page.locator('main.you [data-item]')).toHaveText([/Perfil/, /Central/])
})

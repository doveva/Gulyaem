import { expect, test, type Page } from '@playwright/test'

const EMPTY_SEGMENTS = {
  type: 'FeatureCollection',
  features: [],
  meta: {
    geoDataVersionId: 'e2e-version',
    returnedCount: 0,
    bbox: [30.3, 59.93, 30.33, 59.945],
    statistics: {
      segmentsTotal: 0,
      exploreCount: 0,
      routableOnlyCount: 0,
      ignoreCount: 0,
      totalLengthMeters: 0,
      explorableLengthMeters: 0,
      minLengthMeters: 0,
      medianLengthMeters: 0,
      p95LengthMeters: 0,
      maxLengthMeters: 0,
      shortSegmentCount: 0,
      longSegmentCount: 0,
    },
  },
}

async function useDeterministicStyle(page: Page) {
  await page.route('https://tiles.openfreemap.org/styles/liberty**', async (route) => {
    await route.fulfill({ json: { version: 8, sources: {}, layers: [] } })
  })
}

async function clickRenderedSegment(page: Page) {
  await expect.poll(async () => {
    return page.evaluate(() => {
      const map = window.__GULYAEM_DEBUG_MAP__
      if (!map?.loaded()) return false
      const layers = ['segments-explore', 'segments-routable', 'segments-ignore']
        .filter((layer) => map.getLayer(layer))
      for (let y = 110; y < 760; y += 3) {
        for (let x = 390; x < 900; x += 3) {
          if (map.queryRenderedFeatures([x, y], { layers }).length === 0) continue
          const canvas = map.getCanvas()
          const bounds = canvas.getBoundingClientRect()
          canvas.dispatchEvent(new MouseEvent('click', {
            bubbles: true,
            clientX: bounds.left + x,
            clientY: bounds.top + y,
          }))
          return true
        }
      }
      return false
    })
  }, { timeout: 10_000 }).toBe(true)
}

test.beforeEach(async ({ page }) => {
  await useDeterministicStyle(page)
})

test('covers the core playground interaction flow', async ({ page }) => {
  const initialSegments = page.waitForResponse((response) =>
    response.url().includes('/api/v1/geo/segments?') && response.status() === 200)
  await page.goto('/debug/geo')
  await initialSegments

  await expect(page.getByRole('status')).toContainText('READY')
  await expect(page.getByLabel('Загружено')).toBeVisible()
  await expect(page.locator('.control-panel .stats-grid').last().locator('strong').first()).not.toHaveText('0')

  const detailResponse = page.waitForResponse((response) =>
    /\/api\/v1\/geo\/segments\/[0-9a-f-]+(?:\?|$)/.test(response.url()) && response.status() === 200)
  await clickRenderedSegment(page)
  await detailResponse
  await expect(page.getByText('Нормализация сегмента')).toBeVisible()
  await page.getByLabel('Закрыть инспектор').click()

  const layerReload = page.waitForResponse((response) => {
    if (!response.url().includes('/api/v1/geo/segments?')) return false
    const classifications = new URL(response.url()).searchParams.get('classification') ?? ''
    return !classifications.includes('EXPLORE')
  })
  await page.getByRole('checkbox', { name: /Исследуемые/ }).uncheck()
  await layerReload

  await page.getByLabel('Минимальная длина').fill('20')
  await page.getByLabel('Максимальная длина').fill('200')
  const filterReload = page.waitForResponse((response) => {
    if (!response.url().includes('/api/v1/geo/segments?')) return false
    const query = new URL(response.url()).searchParams
    return query.get('minLength') === '20' && query.get('maxLength') === '200'
  })
  await page.getByRole('button', { name: 'Применить' }).click()
  await filterReload

  const viewportReload = page.waitForResponse((response) =>
    response.url().includes('/api/v1/geo/segments?') && response.status() === 200)
  await page.getByLabel('Тестовый маршрут').selectOption('akademicheskaya-grazhdansky')
  await viewportReload

  await expect(page.getByLabel('Профиль')).toHaveValue('balanced')
  const analysisResponse = page.waitForResponse((response) =>
    response.url().includes('/akademicheskaya-grazhdansky/analyze') && response.status() === 200)
  await page.getByRole('button', { name: 'Анализировать' }).click()
  const analysisPayload = await (await analysisResponse).json()
  expect(analysisPayload.coverageProfile.name).toBe('balanced')
  expect(analysisPayload.coverageProfile.radiusMeters).toBe(100)
  expect(analysisPayload.coverageProfile.coverageRatio).toBe(0.4)
  expect(analysisPayload.contextRadiusMeters).toBe(225)
  await expect(page.getByText('Completed', { exact: true }).first()).toBeVisible()
  await expect(page.getByText('Matched', { exact: true })).toBeVisible()

  await page.getByLabel('Сравнить routing engines').check()
  await expect(page.getByText('Waypoints: 3 · corridor 20 м')).toBeVisible()
  await page.getByLabel('OSRM', { exact: true }).uncheck()
  await expect(page.getByLabel('OSRM', { exact: true })).not.toBeChecked()
})

test('shows viewport API errors', async ({ page }) => {
  await page.route('**/api/v1/geo/segments?**', async (route) => {
    await route.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({ error: { code: 'e2e_failure', message: 'E2E viewport failure' } }),
    })
  })
  await page.goto('/debug/geo')
  await expect(page.getByText('Viewport не загружен')).toBeVisible()
  await expect(page.getByText('E2E viewport failure')).toBeVisible()
  await expect(page.locator('.control-panel .stats-grid').last().locator('strong').first()).toHaveText('0')
})

test('renders an empty dataset without treating it as an error', async ({ page }) => {
  await page.route('**/api/v1/geo/segments?**', async (route) => {
    await route.fulfill({ json: EMPTY_SEGMENTS })
  })
  await page.goto('/debug/geo')
  await expect(page.getByLabel('Загружено')).toBeVisible()
  await expect(page.getByText('Viewport не загружен')).toHaveCount(0)
  await expect(page.locator('.control-panel .stats-grid').last().locator('strong').first()).toHaveText('0')
})

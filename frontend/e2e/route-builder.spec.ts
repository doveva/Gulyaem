import { expect, test, type Page, type Route } from '@playwright/test'

const EMPTY_SEGMENTS = {
  type: 'FeatureCollection', features: [],
  meta: {
    geoDataVersionId: 'e2e-version', returnedCount: 0, bbox: [30.3, 59.93, 30.33, 59.945],
    statistics: {
      segmentsTotal: 0, exploreCount: 0, routableOnlyCount: 0, ignoreCount: 0,
      totalLengthMeters: 0, explorableLengthMeters: 0, minLengthMeters: 0,
      medianLengthMeters: 0, p95LengthMeters: 0, maxLengthMeters: 0,
      shortSegmentCount: 0, longSegmentCount: 0,
    },
  },
}

test.beforeEach(async ({ page }) => {
  await page.route('https://tiles.openfreemap.org/styles/liberty**', async (route) => {
    await route.fulfill({ json: { version: 8, sources: {}, layers: [] } })
  })
  await page.route('**/api/v1/geo/segments?**', async (route) => route.fulfill({ json: EMPTY_SEGMENTS }))
})

test('builds and edits a potential exploration preview', async ({ page }) => {
  const waypointCounts: number[] = []
  await page.route('**/api/v1/route-previews', async (route) => {
    const request = route.request().postDataJSON() as { waypoints: unknown[] }
    waypointCounts.push(request.waypoints.length)
    await fulfillPreview(route, request.waypoints.length)
  })
  await page.goto('/map')
  await page.getByRole('button', { name: '+ Создать прогулку' }).click()
  await clickMap(page, 760, 330)
  await clickMap(page, 940, 470)
  await expect(page.getByText('4,2 км')).toBeVisible()
  await expect(page.getByText('27').first()).toBeVisible()
  await expect(page.getByText('Потенциально исследуется')).toBeVisible()
  await expect(page.locator('.route-marker')).toHaveCount(2)

  await page.getByRole('button', { name: '+ Точка' }).click()
  await clickMap(page, 850, 390)
  await expect.poll(() => waypointCounts.at(-1)).toBe(3)
  await expect(page.locator('.route-marker')).toHaveCount(3)
  await page.getByRole('button', { name: 'Удалить точку 1' }).click()
  await expect.poll(() => waypointCounts.at(-1)).toBe(2)
  await expect(page.locator('.route-marker')).toHaveCount(2)
})

test('keeps waypoints editable after a no-route error', async ({ page }) => {
  await page.route('**/api/v1/route-previews', async (route) => {
    await route.fulfill({
      status: 422,
      contentType: 'application/json',
      body: JSON.stringify({ error: { code: 'route_not_found', message: 'pedestrian route could not be built' } }),
    })
  })
  await page.goto('/map')
  await page.getByRole('button', { name: '+ Создать прогулку' }).click()
  await clickMap(page, 760, 330)
  await clickMap(page, 940, 470)
  await expect(page.getByRole('alert')).toContainText('Не удалось построить пешеходный маршрут')
  await expect(page.locator('.route-marker')).toHaveCount(2)
  await expect(page.getByRole('button', { name: 'Очистить' })).toBeEnabled()
})

test('keeps the two-point builder usable on a mobile viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.route('**/api/v1/route-previews', async (route) => fulfillPreview(route, 2))
  await page.goto('/map')
  await page.getByRole('button', { name: '+ Создать прогулку' }).click()
  await clickMap(page, 220, 180)
  await clickMap(page, 285, 305)
  await expect(page.getByText('4,2 км')).toBeVisible()
  await expect(page.getByRole('button', { name: '+ Точка' })).toBeVisible()
  await expect(page.locator('.route-marker')).toHaveCount(2)
})

async function clickMap(page: Page, x: number, y: number) {
  await expect(page.getByLabel('Карта для построения прогулки')).toBeVisible()
  await page.mouse.click(x, y)
}

async function fulfillPreview(route: Route, waypointCount: number) {
  await route.fulfill({
    json: {
      geoDataVersion: {
        id: '02900000-0000-7000-8000-000000000001', cityId: '01900000-0000-7000-8000-000000000001',
        sourceChecksum: 'e2e', normalizationVersion: 'stage1-segments-v1', status: 'READY', importedAt: null,
      },
      routing: {
        engine: 'valhalla', profile: 'pedestrian', distanceMeters: 4200, durationSeconds: 3600,
        geometry: { type: 'LineString', coordinates: [[30.31, 59.935], [30.325, 59.94]] },
        waypoints: Array.from({ length: waypointCount }, () => ({ input: { lat: 59.935, lon: 30.31 } })),
      },
      explorationPreview: {
        coverageProfile: { name: 'balanced', radiusMeters: 50, coverageRatio: .6, minRequiredMeters: 15, maxRequiredMeters: 80 },
        normalizedRoute: { type: 'MultiLineString', coordinates: [[[30.31, 59.935], [30.325, 59.94]]] },
        matchedFragments: [], unmatchedFragments: [], coverageSegments: [],
        metrics: {
          routeMatchedRatio: .98, routeUnmatchedLengthMeters: 20, completedNetworkLengthMeters: 1800,
          contextExplorableLengthMeters: 9600, completedNetworkRatio: .18,
          completedSegmentCount: 27, partialSegmentCount: 6,
          matchedExplorableRouteLengthMeters: 4000, matchedRoutableOnlyRouteLengthMeters: 180,
        },
      },
      warnings: [],
    },
  })
}

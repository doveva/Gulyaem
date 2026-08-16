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
  await page.route('**/api/v1/cities/*/exploration/segments?**', async (route) => route.fulfill({ json: { type: 'FeatureCollection', features: [] } }))
  await page.route('**/api/v1/cities/*/exploration', async (route) => route.fulfill({ json: {
    geoDataVersion: { id: '02900000-0000-7000-8000-000000000001' }, state: { status: 'READY' },
    city: { exploredLengthMeters: 0, eligibleLengthMeters: 10000, percentage: 0, exploredSegmentsCount: 0 }, districts: [],
  } }))
})

test('completes the first Walk and shows a persistent exploration reward', async ({ page }) => {
	await page.route('**/api/v1/route-previews', async (route) => fulfillPreview(route, 2))
	await mockWalkBackend(page, [3])
	await page.goto('/map')
	await page.getByRole('button', { name: '+ Создать прогулку' }).click()
	await clickMap(page, 760, 330); await clickMap(page, 940, 470)
  await page.getByRole('button', { name: 'Начать прогулку' }).click()
  await expect(page.getByText('Идём по маршруту')).toBeVisible()
  await page.getByRole('button', { name: 'Завершить и проверить' }).click()
  await expect(page.getByText('Маршрут прогулки')).toBeVisible()
  await page.getByRole('button', { name: 'Подтвердить исследование' }).click()
  await expect(page.getByText('Новые улицы открыты!')).toBeVisible()
	await expect(page.getByText('1,8 км')).toBeVisible()
})

test('corrects a REVIEW route, saves it, and completes the corrected Walk', async ({ page }) => {
	await page.route('**/api/v1/route-previews', async (route) => {
		const request = route.request().postDataJSON() as { waypoints: APIWaypoint[] }
		await fulfillPreview(route, request.waypoints.length)
	})
	const backend = await mockWalkBackend(page, [2])
	await page.goto('/map')
	await page.getByRole('button', { name: '+ Создать прогулку' }).click()
	await clickMap(page, 760, 330); await clickMap(page, 940, 470)
	await page.getByRole('button', { name: 'Начать прогулку' }).click()
	await page.getByRole('button', { name: 'Завершить и проверить' }).click()
	await expect(page.getByText('Маршрут прогулки')).toBeVisible()

	await page.getByRole('button', { name: '+ Точка' }).click()
	await clickMap(page, 850, 390)
	await expect(page.locator('.route-marker')).toHaveCount(3)
	const save = page.getByRole('button', { name: 'Сохранить исправление' })
	await expect(save).toBeEnabled()
	await save.click()
	await expect.poll(() => backend.corrections.length).toBe(1)
	expect(backend.corrections[0].waypoints).toHaveLength(3)
	expect(backend.walks[0].revision).toBe(2)

	const complete = page.getByRole('button', { name: 'Подтвердить исследование' })
	await expect(complete).toBeEnabled()
	await complete.click()
	await expect(page.getByText('Новые улицы открыты!')).toBeVisible()
})

test('shows a valid zero-new summary when a second Walk repeats the route', async ({ page }) => {
	await page.route('**/api/v1/route-previews', async (route) => fulfillPreview(route, 2))
	const backend = await mockWalkBackend(page, [3, 0])
	await page.goto('/map')

	await completeWalkFromMap(page)
	await expect(page.getByText('Новые улицы открыты!')).toBeVisible()
	await page.getByRole('button', { name: 'Вернуться к карте' }).click()

	await completeWalkFromMap(page)
	await expect(page.getByText('Знакомый маршрут')).toBeVisible()
	await expect(page.getByText('0', { exact: true })).toBeVisible()
	await expect(page.getByText('0 м')).toBeVisible()
	expect(backend.walks).toHaveLength(2)
})

test('saves a DRAFT explicitly and starts the persisted Walk later', async ({ page }) => {
	await page.route('**/api/v1/route-previews', async (route) => fulfillPreview(route, 2))
	const backend = await mockWalkBackend(page, [1])
	await page.goto('/map')
	await page.getByRole('button', { name: '+ Создать прогулку' }).click()
	await clickMap(page, 760, 330); await clickMap(page, 940, 470)
	await page.getByRole('button', { name: 'Сохранить черновик' }).click()
	await expect(page.getByRole('heading', { name: 'Прогулка готова' })).toBeVisible()
	expect(backend.walks).toHaveLength(1)
	await page.getByRole('button', { name: 'Начать сохранённую прогулку' }).click()
	await expect(page.getByText('Идём по маршруту')).toBeVisible()
})

test('shows persisted district coverage and refreshes it on demand', async ({ page }) => {
	let explorationReads = 0
	await page.route('**/api/v1/cities/*/exploration', async (route) => {
		explorationReads += 1
		await route.fulfill({ json: {
			geoDataVersion: { id: '02900000-0000-7000-8000-000000000001' }, state: { status: 'READY' },
			city: { exploredLengthMeters: 1200, eligibleLengthMeters: 10000, percentage: .12, exploredSegmentsCount: 8 },
			districts: [{ districtId: '04900000-0000-7000-8000-000000000001', name: 'Центральный район', exploredLengthMeters: 1200, eligibleLengthMeters: 4800, percentage: .25 }],
		} })
	})
	await page.goto('/map')
	await expect(page.getByText('Центральный район')).toBeVisible()
	await expect(page.getByText('25.0% · 1,2 км')).toBeVisible()
	await page.getByRole('button', { name: 'Обновить прогресс' }).click()
	await expect.poll(() => explorationReads).toBeGreaterThanOrEqual(2)
})

test('restores an ACTIVE Walk from durable local state after reload', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('gulyaem.activeWalkId', '03900000-0000-7000-8000-000000000001'))
  await page.route('**/api/v1/walks/03900000-0000-7000-8000-000000000001', async (route) => route.fulfill({ json: walkAggregate('ACTIVE') }))
  await page.goto('/map')
  await expect(page.getByText('Идём по маршруту')).toBeVisible()
  await page.reload()
  await expect(page.getByText('Идём по маршруту')).toBeVisible()
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

async function completeWalkFromMap(page: Page) {
	await page.getByRole('button', { name: '+ Создать прогулку' }).click()
	await clickMap(page, 760, 330); await clickMap(page, 940, 470)
	await page.getByRole('button', { name: 'Начать прогулку' }).click()
	await page.getByRole('button', { name: 'Завершить и проверить' }).click()
	await expect(page.getByText('Маршрут прогулки')).toBeVisible()
	await page.getByRole('button', { name: 'Подтвердить исследование' }).click()
}

async function fulfillPreview(route: Route, waypointCount: number) {
  await route.fulfill({
    json: {
      previewFingerprint: 'stage3-preview-fingerprint-v1:sha256:e2e',
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
        coverageProfile: { name: 'balanced', radiusMeters: 100, coverageRatio: .4, minRequiredMeters: 15, maxRequiredMeters: 80 },
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

type APIWaypoint = { lat: number; lon: number }

type MockWalkState = {
	sequence: number
	id: string
	routeID: string
	waypoints: APIWaypoint[]
	revision: number
}

const DEFAULT_WAYPOINTS: APIWaypoint[] = [{ lat: 59.935, lon: 30.31 }, { lat: 59.94, lon: 30.325 }]

function walkAggregate(status: 'DRAFT' | 'ACTIVE' | 'REVIEW', state?: Partial<MockWalkState>) {
	const walkID = state?.id ?? '03900000-0000-7000-8000-000000000001'
	const routeID = state?.routeID ?? '03900000-0000-7000-8000-000000000002'
	const waypoints = state?.waypoints ?? DEFAULT_WAYPOINTS
	return {
		walk: { id: walkID, cityId: '01900000-0000-7000-8000-000000000001', routeId: routeID, status,
			...(status !== 'DRAFT' ? { startedAt: '2026-08-12T12:00:00Z' } : {}), ...(status === 'REVIEW' ? { finishedAt: '2026-08-12T13:00:00Z' } : {}) },
		route: { id: routeID, cityId: '01900000-0000-7000-8000-000000000001', geoDataVersionId: '02900000-0000-7000-8000-000000000001', profile: 'pedestrian',
			waypoints, geometry: { type: 'LineString', coordinates: [[30.31, 59.935], [30.325, 59.94]] }, distanceMeters: 4200, estimatedDurationSeconds: 3600, revision: state?.revision ?? 1 },
	}
}

function completion(newSegmentsCount: number, state?: MockWalkState) {
	return { walk: { ...walkAggregate('REVIEW', state).walk, status: 'COMPLETED', completedAt: '2026-08-12T13:01:00Z', durationSeconds: 3600, distanceMeters: 4200 }, exploration: {
		geoDataVersionId: '02900000-0000-7000-8000-000000000001', newSegmentsCount, revisitedSegmentsCount: 2,
		newNetworkLengthMeters: newSegmentsCount ? 1800 : 0, newSegments: { type: 'FeatureCollection', features: [] },
		districts: newSegmentsCount ? [{ districtId: '04900000-0000-7000-8000-000000000001', name: 'Центральный район', percentageBefore: .1, percentageAfter: .12, newLengthMeters: 1800 }] : [],
	} }
}

async function mockWalkBackend(page: Page, newSegmentsByWalk: number[]) {
	const walks: MockWalkState[] = []
	const corrections: Array<{ waypoints: APIWaypoint[] }> = []
	await page.route('**/api/v1/walks', async (route) => {
		const request = route.request().postDataJSON() as { waypoints: APIWaypoint[] }
		const sequence = walks.length
		const suffix = String(sequence + 1).padStart(12, '0')
		const state: MockWalkState = {
			sequence,
			id: `03900000-0000-7000-8000-${suffix}`,
			routeID: `04900000-0000-7000-8000-${suffix}`,
			waypoints: request.waypoints,
			revision: 1,
		}
		walks.push(state)
		await route.fulfill({ status: 201, json: walkAggregate('DRAFT', state) })
	})
	await page.route('**/api/v1/walks/*/start', async (route) => {
		await route.fulfill({ json: walkAggregate('ACTIVE', requireWalk(route, walks)) })
	})
	await page.route('**/api/v1/walks/*/finish', async (route) => {
		await route.fulfill({ json: walkAggregate('REVIEW', requireWalk(route, walks)) })
	})
	await page.route('**/api/v1/walks/*/route', async (route) => {
		const state = requireWalk(route, walks)
		const request = route.request().postDataJSON() as { waypoints: APIWaypoint[] }
		corrections.push(request)
		state.waypoints = request.waypoints
		state.revision += 1
		await route.fulfill({ json: walkAggregate('REVIEW', state) })
	})
	await page.route('**/api/v1/walks/*/complete', async (route) => {
		const state = requireWalk(route, walks)
		await route.fulfill({ json: completion(newSegmentsByWalk[state.sequence] ?? 0, state) })
	})
	return { walks, corrections }
}

function requireWalk(route: Route, walks: MockWalkState[]) {
	const parts = new URL(route.request().url()).pathname.split('/')
	const walkID = parts.at(-2)
	const state = walks.find((candidate) => candidate.id === walkID)
	if (!state) throw new Error(`unknown mocked Walk ${walkID}`)
	return state
}

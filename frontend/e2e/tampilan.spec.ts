import { join } from 'node:path'

import AxeBuilder from '@axe-core/playwright'

import { expect, test } from './fixtures'
import { openInvite } from './helpers'
import { horizontalOverflow, smallTargets, states } from './states'

const WIDTHS = [320, 375, 768, 1024, 1360, 1440]
const HEIGHT = 900
const WCAG = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22a', 'wcag22aa']

for (const width of WIDTHS) {
  test.describe(`lebar ${width}px`, () => {
    test.use({ viewport: { width, height: HEIGHT } })

    for (const [name, reach] of Object.entries(states)) {
      test(`${name}: axe, tanpa scroll horizontal, kontrol minimal 44x44`, async ({
        page,
        db,
        request,
      }, testInfo) => {
        await reach({ page, db, request })
        // Fokus dari alur di atas dilepas, supaya screenshot dan axe melihat tampilan biasa.
        await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur())

        const shot = join(testInfo.project.outputDir, 'screenshots', `${name}-${width}.png`)
        await page.screenshot({ path: shot, fullPage: true })
        await testInfo.attach(`${name}-${width}.png`, { path: shot, contentType: 'image/png' })

        expect(await horizontalOverflow(page), 'lebar konten melebihi layar').toBe(0)
        expect(await smallTargets(page), 'kontrol lebih kecil dari 44x44 px').toEqual([])

        const axe = await new AxeBuilder({ page }).withTags(WCAG).analyze()
        const violations = axe.violations.map(
          (v) => `${v.id} (${v.impact}): ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`,
        )
        expect(violations, 'pelanggaran axe-core').toEqual([])
      })
    }
  })
}

// Aset Doomsday hanya mulai 1360px, dan di bawahnya tidak diunduh supaya kuota HP tidak terpakai.
for (const width of [320, 375, 768, 1024, 1359, 1360, 1440]) {
  test(`gambar samping di ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: HEIGHT })
    const requested: string[] = []
    page.on('request', (r) => {
      if (r.url().includes('/doomsday/')) {
        requested.push(r.url())
      }
    })
    await openInvite(page)
    await page.mouse.wheel(0, 4000)
    await page.waitForLoadState('networkidle')

    const figures = page.locator('figure').filter({ has: page.locator('img[src^="/doomsday/"]') })
    await expect(figures).toHaveCount(2)
    if (width < 1360) {
      await expect(figures.first()).toBeHidden()
      await expect(figures.last()).toBeHidden()
      expect(requested, 'gambar samping diunduh padahal tersembunyi').toEqual([])
      return
    }
    await expect(figures.first()).toBeVisible()
    await expect(figures.last()).toBeVisible()
    expect(requested.length).toBeGreaterThan(0)
    const loaded = await figures
      .locator('img')
      .evaluateAll((imgs) => imgs.map((i) => (i as HTMLImageElement).naturalWidth > 0))
    expect(loaded).toEqual([true, true])
    await expect(page.getByText('Ilustrasi sementara.', { exact: false })).toBeVisible()
    // Teks alternatif hanya di satu sisi, supaya pembaca layar tidak membacakannya dua kali.
    const alts = await figures
      .locator('img')
      .evaluateAll((imgs) => imgs.map((i) => i.getAttribute('alt')))
    expect(alts.filter((a) => a !== '')).toHaveLength(1)
  })
}

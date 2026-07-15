import { expect, test } from '@playwright/test';

test('creates, controls and publicly tracks a stopwatch', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('/');
  await expect(page.getByText('00:00.00')).toBeVisible();
  await page.getByRole('button', { name: 'Start now' }).click();
  await expect(page).toHaveURL(/\/c\//);
  await expect(page.getByText('running', { exact: true })).toBeVisible();

  const shareTrigger = page.getByRole('button', { name: 'Share' });
  await shareTrigger.click();
  await expect(page.getByRole('dialog', { name: 'Share links' })).toBeVisible();
  const publicURL = await page.getByLabel('Public viewer link').inputValue();
  await page.getByRole('button', { name: 'Copy public link' }).click();
  await expect(page.getByRole('dialog', { name: 'Share links' }).getByRole('status')).toHaveText('Public link copied');
  await page.keyboard.press('Escape');
  await expect(shareTrigger).toBeFocused();

  const viewer = await context.newPage();
  await viewer.goto(publicURL);
  await expect(viewer.getByText('running', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Lap' }).click();
  await expect(page.getByText('#1')).toBeVisible();
  await page.getByRole('button', { name: 'Pause' }).click();
  await expect(viewer.getByText('paused', { exact: true })).toBeVisible();
});

test('adds an independent countdown timer', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Start now' }).click();
  await page.getByRole('button', { name: 'Add clock' }).click();
  await expect(page.getByRole('dialog', { name: 'Add clock' })).toBeVisible();
  await page.getByRole('button', { name: '1 minute' }).click();
  await expect(page.getByText('01:00')).toBeVisible();
  await expect(page.getByText('Stopwatch', { exact: true })).toHaveCount(1);
  await expect(page.getByText('Timer', { exact: true })).toHaveCount(1);
});

test('fits supported viewports with touch-sized controls', async ({ page }) => {
  const viewports = [
    { width: 390, height: 844 },
    { width: 844, height: 390 },
    { width: 768, height: 1024 },
    { width: 1440, height: 900 }
  ];

  for (const viewport of viewports) {
    await page.setViewportSize(viewport);
    await page.goto('/');
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(viewport.width);

    await page.getByRole('button', { name: 'Start now' }).click();
    await expect(page).toHaveURL(/\/c\//);
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(viewport.width);

    const buttonHeights = await page.locator('button:visible').evaluateAll((buttons) =>
      buttons.map((button) => button.getBoundingClientRect().height)
    );
    expect(buttonHeights.every((height) => height >= 44)).toBe(true);
  }
});

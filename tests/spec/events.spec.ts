import { test, expect, type Page } from '@playwright/test';

const ROUTES = {
  page: '/events',
  apiCreate: '/api/events',
};

async function navigateToEvents(page: Page) {
  await page.goto(ROUTES.page);
  await page.waitForLoadState('networkidle');
}

async function createTestEvent(page: Page, environmentId: string, title: string) {
  const res = await page.request.post(ROUTES.apiCreate, {
    data: {
      type: 'system.prune',
      severity: 'info',
      title,
      description: 'playwright test event',
      environmentId,
      metadata: { playwright: true },
    },
  });

  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  const id = body?.data?.id as string | undefined;
  expect(id).toBeTruthy();
  return id!;
}

async function deleteTestEvent(page: Page, id: string) {
  const res = await page.request.delete(`/api/events/${id}`);
  // Best-effort cleanup (events can race in parallel runs)
  expect([200, 404]).toContain(res.status());
}

test.describe('Events Page', () => {
  test('should scope the event list to the selected environment (default env 0)', async ({ page }) => {
    const titleEnv0 = `pw-env0-${Date.now()}`;
    const titleOther = `pw-other-${Date.now()}`;

    const idEnv0 = await createTestEvent(page, '0', titleEnv0);
    const idOther = await createTestEvent(page, '999', titleOther);

    try {
      await navigateToEvents(page);

      // The env-scoped events endpoint should only return env "0" entries by default.
      await expect(page.getByText(titleEnv0).first()).toBeVisible();
      await expect(page.getByText(titleOther).first()).toHaveCount(0);
    } finally {
      await deleteTestEvent(page, idEnv0);
      await deleteTestEvent(page, idOther);
    }
  });
});

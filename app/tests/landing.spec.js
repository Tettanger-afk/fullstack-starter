import { test, expect } from '@playwright/test';

test.describe('Landing Page', () => {
  test('should display PlanIT landing page with title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/PlanIT/);
  });

  test('should display welcome heading', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('PlanIT');
  });

  test('should have navigation links', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('nav')).toContainText('Plans');
    await expect(page.locator('nav')).toContainText('Tasks');
    await expect(page.locator('nav')).toContainText('Dashboard');
  });

  test('should have call-to-action buttons', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('a[href="/tasks/new"]').first()).toBeVisible();
    await expect(page.locator('a[href="/plans/new"]').first()).toBeVisible();
  });
});

test.describe('Dashboard', () => {
  test('should display dashboard page with title', async ({ page }) => {
    await page.goto('/dashboard');
    await expect(page).toHaveTitle(/Dashboard/);
  });

  test('should show task stats', async ({ page }) => {
    await page.goto('/dashboard');
    await expect(page.locator('h2')).toContainText('Dashboard');
    await expect(page.locator('.card')).toHaveCount({ minimum: 1 });
  });
});

test.describe('Plans', () => {
  test('should display plans list page', async ({ page }) => {
    await page.goto('/plans');
    await expect(page).toHaveTitle(/Plans/);
    await expect(page.locator('h2')).toContainText('My Plans');
  });

  test('should display create plan form', async ({ page }) => {
    await page.goto('/plans/new');
    await expect(page).toHaveTitle(/Create Plan/);
    await expect(page.locator('form')).toBeVisible();
    await expect(page.locator('input[name="title"]')).toBeVisible();
  });

  test('should create a new plan', async ({ page }) => {
    await page.goto('/plans/new');
    await page.fill('input[name="title"]', 'Test Plan E2E');
    await page.fill('textarea[name="description"]', 'Created by Playwright');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL('/plans');
    await expect(page.locator('body')).toContainText('Test Plan E2E');
  });
});

test.describe('Tasks', () => {
  test('should display tasks list page', async ({ page }) => {
    await page.goto('/tasks');
    await expect(page).toHaveTitle(/Tasks/);
    await expect(page.locator('h2')).toContainText('My Tasks');
  });

  test('should display create task form', async ({ page }) => {
    await page.goto('/tasks/new');
    await expect(page).toHaveTitle(/Create Task/);
    await expect(page.locator('form')).toBeVisible();
    await expect(page.locator('input[name="title"]')).toBeVisible();
    await expect(page.locator('select[name="plan_id"]')).toBeVisible();
  });

  test('should have filter controls', async ({ page }) => {
    await page.goto('/tasks');
    await expect(page.locator('select[name="status"]')).toBeVisible();
    await expect(page.locator('select[name="category"]')).toBeVisible();
  });
});

test.describe('404 Page', () => {
  test('should return 404 for unknown routes', async ({ page }) => {
    const response = await page.goto('/this-does-not-exist');
    expect(response.status()).toBe(404);
    await expect(page.locator('h1')).toContainText('404');
  });
});

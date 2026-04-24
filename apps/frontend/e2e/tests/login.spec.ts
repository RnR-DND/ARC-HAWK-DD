import { test, expect } from '@playwright/test';

test.describe('Login page', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login');
    });

    test('renders login form with all fields', async ({ page }) => {
        await expect(page.getByTestId('login-form')).toBeVisible();
        await expect(page.getByTestId('login-email')).toBeVisible();
        await expect(page.getByTestId('login-password')).toBeVisible();
        await expect(page.getByTestId('login-tenant-id')).toBeVisible();
        await expect(page.getByTestId('login-submit')).toBeVisible();
    });

    test('submit button text is "Sign in"', async ({ page }) => {
        await expect(page.getByTestId('login-submit')).toHaveText('Sign in');
    });

    test('shows error banner on failed login', async ({ page }) => {
        // Intercept the login API and return 401.
        await page.route('**/api/v1/auth/login', route =>
            route.fulfill({
                status: 401,
                contentType: 'application/json',
                body: JSON.stringify({ message: 'Invalid credentials' }),
            })
        );

        await page.getByTestId('login-email').fill('bad@example.com');
        await page.getByTestId('login-password').fill('wrongpassword');
        await page.getByTestId('login-tenant-id').fill('00000000-0000-0000-0000-000000000001');
        await page.getByTestId('login-submit').click();

        await expect(page.getByTestId('login-error')).toBeVisible();
        await expect(page.getByTestId('login-error')).toContainText('Invalid credentials');
    });

    test('redirects to dashboard on successful login', async ({ page }) => {
        await page.route('**/api/v1/auth/login', route =>
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    access_token: 'test-token-abc',
                    refresh_token: 'test-refresh-abc',
                    token_type: 'Bearer',
                    expires_in: 86400,
                    user: { id: 'u1', email: 'admin@example.com' },
                }),
            })
        );

        await page.getByTestId('login-email').fill('admin@example.com');
        await page.getByTestId('login-password').fill('correctpassword');
        await page.getByTestId('login-tenant-id').fill('00000000-0000-0000-0000-000000000001');
        await page.getByTestId('login-submit').click();

        await expect(page).toHaveURL('/');
    });

    test('stores arc_token in localStorage on success', async ({ page }) => {
        await page.route('**/api/v1/auth/login', route =>
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ access_token: 'stored-token-xyz', token_type: 'Bearer' }),
            })
        );

        await page.getByTestId('login-email').fill('admin@example.com');
        await page.getByTestId('login-password').fill('correctpassword');
        await page.getByTestId('login-tenant-id').fill('00000000-0000-0000-0000-000000000001');
        await page.getByTestId('login-submit').click();

        // Wait for navigation then verify localStorage.
        await page.waitForURL('/');
        const token = await page.evaluate(() => localStorage.getItem('arc_token'));
        expect(token).toBe('stored-token-xyz');
    });

    test('session-expired message shown when redirected from expired session', async ({ page }) => {
        // Simulate the api-client having set the session_expired flag.
        await page.evaluate(() => localStorage.setItem('session_expired', '1'));
        await page.reload();

        await expect(page.getByTestId('login-error')).toContainText('session expired');
    });
});

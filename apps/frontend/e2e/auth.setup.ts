import { test as setup } from '@playwright/test';
import path from 'path';

export const authFile = path.join(__dirname, '.auth/user.json');

// Populates storageState with a valid arc_token by calling the login API.
// Only runs when PLAYWRIGHT_AUTH_EMAIL / _PASSWORD / _TENANT_ID are set.
// If env vars are absent the setup is skipped — tests run in dev/no-auth mode.
setup('authenticate', async ({ request, page }) => {
    const email = process.env.PLAYWRIGHT_AUTH_EMAIL;
    const password = process.env.PLAYWRIGHT_AUTH_PASSWORD;
    const tenantId = process.env.PLAYWRIGHT_AUTH_TENANT_ID;

    if (!email || !password || !tenantId) {
        // No credentials supplied — save empty state so dependent tests still run.
        await page.goto('/');
        await page.context().storageState({ path: authFile });
        return;
    }

    const baseURL = process.env.BASE_URL || 'http://localhost:3000';
    const apiBase = process.env.API_BASE_URL || 'http://localhost:8080/api/v1';

    const res = await request.post(`${apiBase}/auth/login`, {
        data: { email, password, tenant_id: tenantId },
    });

    if (!res.ok()) {
        throw new Error(`Login API returned ${res.status()}: ${await res.text()}`);
    }

    const body = await res.json();
    const token: string = body.access_token;

    // Navigate to the app and inject the token into localStorage.
    await page.goto(baseURL);
    await page.evaluate((t: string) => localStorage.setItem('arc_token', t), token);
    await page.context().storageState({ path: authFile });
});

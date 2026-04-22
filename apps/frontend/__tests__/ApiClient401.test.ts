import '@testing-library/jest-dom';

// sessionExpiredRedirectInFlight is module-level. The only way to reset it
// between tests is to reload the module. We use jest.resetModules() +
// dynamic require in beforeEach so each test starts with the flag = false.

function makeAxiosError(status: number) {
    const err: any = new Error(`Request failed with status code ${status}`);
    err.isAxiosError = true;
    err.response = { status, statusText: 'Error', data: {}, headers: {}, config: {} };
    err.config = { method: 'get', url: '/test' };
    return err;
}

describe('apiClient 401/403 interceptor', () => {
    let client: any;

    beforeEach(() => {
        jest.resetModules();

        // Fresh window.location (jsdom makes href read-only by default)
        delete (window as any).location;
        (window as any).location = { href: '', pathname: '/dashboard', search: '' };

        localStorage.clear();

        // Fresh module import — sessionExpiredRedirectInFlight resets to false
        client = require('../utils/api-client').apiClient;
    });

    function stubReject(status: number) {
        client.defaults.adapter = jest.fn().mockRejectedValue(makeAxiosError(status));
    }

    it('sets session_expired in localStorage on 401', async () => {
        stubReject(401);
        await expect(client.get('/test')).rejects.toBeDefined();
        expect(localStorage.getItem('session_expired')).toBe('1');
    });

    it('redirects to /login with encoded return path on 401', async () => {
        (window as any).location = { href: '', pathname: '/scans', search: '?q=1' };
        stubReject(401);
        await expect(client.get('/test')).rejects.toBeDefined();
        expect((window as any).location.href).toBe('/login?redirect=%2Fscans%3Fq%3D1');
    });

    it('does not redirect when already on /login path', async () => {
        (window as any).location = { href: '', pathname: '/login', search: '' };
        stubReject(401);
        await expect(client.get('/test')).rejects.toBeDefined();
        expect((window as any).location.href).toBe('');
    });

    it('treats 403 the same as 401', async () => {
        stubReject(403);
        await expect(client.get('/test')).rejects.toBeDefined();
        expect(localStorage.getItem('session_expired')).toBe('1');
    });

    it('does not trigger on 500 errors', async () => {
        stubReject(500);
        await expect(client.get('/test')).rejects.toBeDefined();
        expect(localStorage.getItem('session_expired')).toBeNull();
        expect((window as any).location.href).toBe('');
    });

    it('debounce: concurrent 401s only write session_expired once', async () => {
        stubReject(401);
        const spy = jest.spyOn(Storage.prototype, 'setItem');
        await Promise.allSettled([
            client.get('/a'),
            client.get('/b'),
            client.get('/c'),
        ]);
        const hits = spy.mock.calls.filter((c) => c[0] === 'session_expired');
        expect(hits.length).toBe(1);
    });
});

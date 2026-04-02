import axios from 'axios';
import { apiClient } from '@/utils/api-client';

export interface HealthStatus {
    status: 'healthy' | 'unhealthy' | 'degraded';
    service: string;
    timestamp: string;
    dependencies?: Record<string, string>;
}

/**
 * Check backend health status
 * Uses a shorter timeout to fail fast.
 * Derives the base URL from the shared apiClient to avoid independent definitions.
 */
export async function checkBackendHealth(): Promise<HealthStatus> {
    try {
        // Health endpoint is at root /health, outside the /api/v1 prefix
        const baseURL = (apiClient.defaults.baseURL || '').replace(/\/api\/v1\/?$/, '');
        const healthUrl = `${baseURL}/health`;

        const response = await axios.get(healthUrl, { timeout: 2000 });
        return response.data;
    } catch (error) {
        console.error('Backend health check failed', error);
        return {
            status: 'unhealthy',
            service: 'arc-hawk-backend',
            timestamp: new Date().toISOString()
        };
    }
}

export const healthApi = {
    checkBackendHealth
};

export default healthApi;

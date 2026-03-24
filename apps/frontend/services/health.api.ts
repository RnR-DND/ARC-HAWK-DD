import axios from 'axios';
import { apiClient } from '@/utils/api-client';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export interface HealthStatus {
    status: 'healthy' | 'unhealthy' | 'degraded';
    service: string;
    timestamp: string;
    dependencies?: Record<string, string>;
}

/**
 * Check backend health status
 * Uses a shorter timeout to fail fast
 */
export async function checkBackendHealth(): Promise<HealthStatus> {
    try {
        // Health endpoint is typically at root /health or /api/health not /api/v1/health
        // Adjusting path to match typical backend setup: http://localhost:8080/health
        const healthUrl = API_BASE_URL.replace('/api/v1', '') + '/health';

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

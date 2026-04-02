import axios, { AxiosInstance, AxiosResponse, AxiosError, InternalAxiosRequestConfig } from 'axios';

// Standard API Response wrapper
export interface ApiResponse<T> {
    data: T;
    error?: string;
    description?: string;
    details?: string[];
}

const isServer = typeof window === 'undefined';
// Client: use relative path /api/v1 (proxied by Next.js)
// Server: use full Docker URL from env
const API_BASE_URL = isServer
    ? (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1')
    : '/api/v1';

// Create Axios Instance with strict defaults
export const apiClient: AxiosInstance = axios.create({
    baseURL: API_BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
    timeout: 10000, // 10s timeout
});

// Request Interceptor
apiClient.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
        if (typeof window !== 'undefined') {
            const token = localStorage.getItem('arc_hawk_token');
            if (token) {
                config.headers = config.headers || {};
                config.headers.Authorization = `Bearer ${token}`;
            }
        }
        return config;
    },
    (error) => Promise.reject(error)
);

// Response Interceptor
apiClient.interceptors.response.use(
    (response: AxiosResponse) => response,
    async (error: AxiosError) => {
        const originalRequest = error.config;

        // Log Error
        console.error(`API Error: ${error.config?.method?.toUpperCase()} ${error.config?.url}`, {
            status: error.response?.status,
            data: error.response?.data,
            message: error.message
        });

        // Simple Retry Logic (optional, for idempotency)
        // if (error.code === 'ECONNABORTED' && !originalRequest._retry) {
        //     originalRequest._retry = true;
        //     return apiClient(originalRequest);
        // }

        return Promise.reject(error);
    }
);

/**
 * Generic fetcher for creating strictly typed service helpers
 */
export async function get<T>(url: string, params?: any): Promise<T> {
    const response = await apiClient.get<T>(url, { params });
    return response.data;
}

export async function post<T>(url: string, body: any): Promise<T> {
    const response = await apiClient.post<T>(url, body);
    return response.data;
}

export async function put<T>(url: string, body: any): Promise<T> {
    const response = await apiClient.put<T>(url, body);
    return response.data;
}

export async function del<T>(url: string): Promise<T> {
    const response = await apiClient.delete<T>(url);
    return response.data;
}

export default apiClient;

import { get, post, put, del } from '@/utils/api-client';

export interface CustomPattern {
    id?: string;
    name: string;
    display_name: string;
    regex: string;
    category: string;
    description?: string;
    is_active?: boolean;
    created_at?: string;
    context_keywords?: string[];
    negative_keywords?: string[];
}

export const patternsApi = {
    getPatterns: async (): Promise<CustomPattern[]> => {
        try {
            const response = await get<any>('/patterns');
            return response?.data ?? [];
        } catch {
            return [];
        }
    },

    createPattern: async (pattern: Omit<CustomPattern, 'id' | 'created_at'>): Promise<CustomPattern> => {
        const response = await post<any>('/patterns', pattern);
        return response?.data ?? response;
    },

    updatePattern: async (id: string, pattern: Partial<CustomPattern>): Promise<CustomPattern> => {
        const response = await put<any>(`/patterns/${id}`, pattern);
        return response?.data ?? response;
    },

    deletePattern: async (id: string): Promise<void> => {
        await del<any>(`/patterns/${id}`);
    },
};

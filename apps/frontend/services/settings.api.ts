import { post, get, put } from '@/utils/api-client';
import { UserSettings } from '@/types';

export const settingsApi = {
    /**
     * Get system settings
     */
    getSettings: async (): Promise<UserSettings | null> => {
        try {
            const response = await get<UserSettings>('/auth/settings');
            return response;
        } catch (error) {
            console.error('Failed to fetch settings:', error);
            return null;
        }
    },

    /**
     * Update system settings
     */
    updateSettings: async (settings: UserSettings): Promise<UserSettings> => {
        return await put<UserSettings>('/auth/settings', { settings });
    }
};

export default settingsApi;

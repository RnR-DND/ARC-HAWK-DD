import { get } from '@/utils/api-client';
import { User } from '@/types/api';

export const authApi = {
    getProfile: async (): Promise<User> => {
        return get<User>('/auth/profile');
    },
};

export default authApi;

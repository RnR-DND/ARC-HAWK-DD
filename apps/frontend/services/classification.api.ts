import { get } from '@/utils/api-client';
import { ClassificationSummary } from '@/types';

export const classificationApi = {
    getSummary: async (): Promise<ClassificationSummary> => {
        // Backend usually returns wrapped data
        const res = await get<{ data: ClassificationSummary }>('/classification/summary');
        return res.data;
    },
};

export default classificationApi;

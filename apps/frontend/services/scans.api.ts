import { post, get, del } from '@/utils/api-client';
import { IngestResult } from '@/types';

export const scansApi = {
    /**
     * Ingest a new scan result
     */
    ingestScan: async (scanData: any): Promise<IngestResult> => {
        // Backend returns wrapped data? Let's assume standard wrapper
        // and robustly handle it inside get/post if possible, but 
        // our new api-client returns .data directly.
        // Let's stick to the convention of explicit typing.
        return await post<IngestResult>('/scans/ingest-verified', scanData);
    },

    triggerScan: async (config: any): Promise<any> => {
        return await post<any>('/scans/trigger', config);
    },

    /**
     * Get the last scan run details
     */
    getLastScanRun: async (): Promise<any> => {
        try {
            const response = await get<any>('/scans/latest');
            // Backend returns { data: {...} } wrapper
            return response?.data ?? response;
        } catch (error) {
            console.error('Failed to fetch last scan:', error);
            return null;
        }
    },

    getScans: async (): Promise<any[]> => {
        try {
            // The backend returns { data: [...] } structure
            const response = await get<any>('/scans');
            // Unwrap the backend's response wrapper and handle null
            const scans = response?.data ?? response;
            return Array.isArray(scans) ? scans : [];
        } catch (error) {
            console.error('Failed to fetch scans:', error);
            return [];
        }
    },

    getScan: async (id: string): Promise<any> => {
        try {
            const response = await get<any>(`/scans/${id}`);
            return response;
        } catch (error) {
            console.error(`Failed to fetch scan ${id}:`, error);
            throw error;
        }
    },

    getScanStatus: async (id: string): Promise<any> => {
        try {
            return await get<any>(`/scans/${id}/status`);
        } catch (error) {
            console.error(`Failed to fetch scan status ${id}:`, error);
            throw error;
        }
    },

    cancelScan: async (id: string): Promise<any> => {
        return await post<any>(`/scans/${id}/cancel`, {});
    },

    clearScanData: async (): Promise<any> => {
        return await del<any>('/scans/clear');
    },

    deleteScan: async (id: string): Promise<any> => {
        return await del<any>(`/scans/${id}`);
    },

    getScanPIISummary: async (id: string): Promise<any[]> => {
        try {
            const response = await get<any>(`/scans/${id}/pii-summary`);
            return response?.data ?? [];
        } catch (error) {
            console.error(`Failed to fetch PII summary for scan ${id}:`, error);
            return [];
        }
    }
};

export default scansApi;

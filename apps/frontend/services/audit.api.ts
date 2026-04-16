import { get } from '@/utils/api-client';

type AuditLogResponse = AuditLogEntry[] | { data?: AuditLogEntry[]; logs?: AuditLogEntry[] };

export interface AuditLogEntry {
    id: string;
    user_id: string;
    action: string;
    resource_type: string;
    resource_id: string;
    ip_address?: string;
    result?: string;
    metadata?: Record<string, unknown>;
    event_time: string;
}

export interface AuditLogFilters {
    user_id?: string;
    action?: string;
    resource_type?: string;
    start_time?: string;
    end_time?: string;
    limit?: number;
    offset?: number;
}

export const auditApi = {
    getLogs: async (filters?: AuditLogFilters): Promise<AuditLogEntry[]> => {
        try {
            const params = new URLSearchParams();
            if (filters?.user_id)       params.set('user_id', filters.user_id);
            if (filters?.action)        params.set('action', filters.action);
            if (filters?.resource_type) params.set('resource_type', filters.resource_type);
            if (filters?.start_time)    params.set('start_time', filters.start_time);
            if (filters?.end_time)      params.set('end_time', filters.end_time);
            if (filters?.limit)         params.set('limit', String(filters.limit));
            if (filters?.offset)        params.set('offset', String(filters.offset));

            const query = params.toString() ? `?${params}` : '';
            const res = await get<AuditLogResponse>(`/audit/logs${query}`);
            return Array.isArray(res) ? res : (res?.data ?? res?.logs ?? []);
        } catch {
            return [];
        }
    },

    getRecentActivity: async (limit = 20): Promise<AuditLogEntry[]> => {
        try {
            const res = await get<AuditLogResponse>(`/audit/recent?limit=${limit}`);
            return Array.isArray(res) ? res : (res?.data ?? res?.logs ?? []);
        } catch {
            return [];
        }
    },

    getUserActivity: async (userId: string): Promise<AuditLogEntry[]> => {
        try {
            const res = await get<AuditLogResponse>(`/audit/user/${userId}`);
            return Array.isArray(res) ? res : (res?.data ?? res?.logs ?? []);
        } catch {
            return [];
        }
    },

    getResourceHistory: async (resourceType: string, resourceId: string): Promise<AuditLogEntry[]> => {
        try {
            const res = await get<AuditLogResponse>(`/audit/resource/${resourceType}/${resourceId}`);
            return Array.isArray(res) ? res : (res?.data ?? res?.logs ?? []);
        } catch {
            return [];
        }
    },
};

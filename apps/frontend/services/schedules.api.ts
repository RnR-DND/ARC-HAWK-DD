import { get, post, del } from '@/utils/api-client';
import apiClient from '@/utils/api-client';

export interface ScanSchedule {
    id: string;
    profile_name: string;
    frequency: 'daily' | 'weekly' | 'monthly';
    hour: number;
    day_of_week?: number;
    day_of_month?: number;
    enabled: boolean;
    last_run_at: string | null;
    next_run_at: string | null;
    created_at: string;
}

export interface CreateSchedulePayload {
    profile_name: string;
    frequency: 'daily' | 'weekly' | 'monthly';
    hour: number;
    day_of_week?: number;
    day_of_month?: number;
}

export async function getSchedules(): Promise<ScanSchedule[]> {
    const res = await get<{ schedules: ScanSchedule[] }>('/scans/schedules');
    return res.schedules ?? [];
}

export async function createSchedule(payload: CreateSchedulePayload): Promise<{ id: string }> {
    return post<{ id: string }>('/scans/schedules', payload);
}

export async function deleteSchedule(id: string): Promise<void> {
    await del<unknown>(`/scans/schedules/${id}`);
}

export async function toggleSchedule(id: string): Promise<{ enabled: boolean }> {
    const res = await apiClient.patch<{ enabled: boolean }>(`/scans/schedules/${id}/toggle`);
    return res.data;
}

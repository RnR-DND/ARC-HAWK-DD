import { get, put } from '@/utils/api-client';

export interface NotificationSettings {
    id?: string;
    email_enabled: boolean;
    email_recipients: string[];
    slack_enabled: boolean;
    slack_webhook_url: string;
    notify_on_scan_complete: boolean;
    notify_on_high_severity: boolean;
    notify_on_stale_connector: boolean;
    severity_threshold: 'Critical' | 'High' | 'Medium' | 'Low';
}

export async function getNotificationSettings(): Promise<NotificationSettings> {
    const res = await get<{ settings: NotificationSettings }>('/auth/settings/notifications');
    return res.settings;
}

export async function saveNotificationSettings(settings: Omit<NotificationSettings, 'id'>): Promise<void> {
    await put<unknown>('/auth/settings/notifications', settings);
}

import { get, post, del } from '@/utils/api-client';

export interface ConnectionConfig {
    source_type: string;
    profile_name: string;
    config: {
        host?: string;
        user?: string;
        password?: string;
        database?: string;
        environment?: string;
        [key: string]: unknown;
    };
}

export interface Connection {
    id: string;
    source_type: string;
    profile_name: string;
    validation_status: string;
    created_at: string;
    updated_at: string;
}

export interface AvailableSourceType {
    source_type: string;
    display_name: string;
    category: string;
    icon: string;
}

/** Normalize config keys: rename username → user for backend compatibility */
function normalizeConfig(data: ConnectionConfig): ConnectionConfig {
    const { config } = data;
    if ('username' in config) {
        const { username, ...rest } = config as { username?: string; [key: string]: unknown };
        return { ...data, config: { ...rest, user: username } };
    }
    return data;
}

export async function addConnection(data: ConnectionConfig): Promise<unknown> {
    return post<unknown>('/connections', normalizeConfig(data));
}

export async function getConnections(): Promise<{ connections: Connection[] }> {
    return get<{ connections: Connection[] }>('/connections');
}

export async function deleteConnection(id: string): Promise<unknown> {
    return del<unknown>(`/connections/${id}`);
}

export async function syncConnections(): Promise<unknown> {
    return post<unknown>('/connections/sync', {});
}

export async function validateSync(): Promise<unknown> {
    return get<unknown>('/connections/sync/validate');
}

export async function testConnection(data: ConnectionConfig): Promise<unknown> {
    return post<unknown>('/connections/test', normalizeConfig(data));
}

export async function getAvailableTypes(): Promise<{ types: AvailableSourceType[] }> {
    return get<{ types: AvailableSourceType[] }>('/connections/available-types');
}

export const connectionsApi = {
    addConnection,
    getConnections,
    deleteConnection,
    syncConnections,
    validateSync,
    testConnection,
    getAvailableTypes,
};

export default connectionsApi;

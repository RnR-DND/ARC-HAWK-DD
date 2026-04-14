/**
 * API utility helpers for consistent response handling across service files.
 *
 * Problem: Backend modules are inconsistent — some wrap responses in {data: ...},
 * others return the object directly. This caused null crashes when the shape changed.
 *
 * Solution: Use unwrapResponse() everywhere instead of ad-hoc `response?.data ?? response`.
 */

/**
 * Safely unwrap API responses that may be wrapped in {data: ...} or returned directly.
 *
 * @example
 *   const data = unwrapResponse(response, []);       // array fallback
 *   const data = unwrapResponse(response, null);     // nullable fallback
 *   const data = unwrapResponse(response, {});       // object fallback
 */
export function unwrapResponse<T>(response: any, fallback: T): T {
    if (response === null || response === undefined) return fallback;
    if (response.data !== undefined) return response.data as T;
    return response as T;
}

/**
 * Safely unwrap an array response, returning an empty array on failure.
 * Handles both {data: [...]} and [...] shapes.
 */
export function unwrapArray<T>(response: any): T[] {
    const result = unwrapResponse<T[]>(response, []);
    return Array.isArray(result) ? result : [];
}

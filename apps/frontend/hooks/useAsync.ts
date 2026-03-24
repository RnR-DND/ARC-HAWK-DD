import { useState, useCallback } from 'react';

interface AsyncState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

interface UseAsyncReturn<T> extends AsyncState<T> {
  execute: (...args: any[]) => Promise<T | null>;
  reset: () => void;
}

/**
 * Hook for handling async operations with loading states and error handling
 * @param asyncFn - The async function to execute
 * @param options - Configuration options
 */
export function useAsync<T>(
  asyncFn: (...args: any[]) => Promise<T>,
  options: {
    onSuccess?: (data: T) => void;
    onError?: (error: string) => void;
    immediate?: boolean;
  } = {}
): UseAsyncReturn<T> {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: false,
    error: null,
  });

  const execute = useCallback(async (...args: any[]): Promise<T | null> => {
    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const data = await asyncFn(...args);
      setState({ data, loading: false, error: null });
      options.onSuccess?.(data);
      return data;
    } catch (error: any) {
      const errorMessage = error?.message || 'An unexpected error occurred';
      setState(prev => ({ ...prev, loading: false, error: errorMessage }));
      options.onError?.(errorMessage);
      return null;
    }
  }, [asyncFn, options]);

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null });
  }, []);

  return {
    ...state,
    execute,
    reset,
  };
}

/**
 * Hook for handling multiple async operations
 */
export function useAsyncMultiple() {
  const [operations, setOperations] = useState<Record<string, AsyncState<any>>>({});

  const createOperation = useCallback(<T,>(
    key: string,
    asyncFn: (...args: any[]) => Promise<T>
  ) => {
    const execute = async (...args: any[]): Promise<T | null> => {
      setOperations(prev => ({
        ...prev,
        [key]: { data: null, loading: true, error: null }
      }));

      try {
        const data = await asyncFn(...args);
        setOperations(prev => ({
          ...prev,
          [key]: { data, loading: false, error: null }
        }));
        return data;
      } catch (error: any) {
        const errorMessage = error?.message || 'An unexpected error occurred';
        setOperations(prev => ({
          ...prev,
          [key]: { data: null, loading: false, error: errorMessage }
        }));
        return null;
      }
    };

    const reset = () => {
      setOperations(prev => ({
        ...prev,
        [key]: { data: null, loading: false, error: null }
      }));
    };

    return {
      execute,
      reset,
      get state() {
        return operations[key] || { data: null, loading: false, error: null };
      }
    };
  }, [operations]);

  return { createOperation };
}
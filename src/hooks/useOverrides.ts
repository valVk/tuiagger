import { useState, useCallback, useEffect } from 'react';
import {
  loadOverrides,
  getEndpointOverride,
  saveEndpointOverride,
  deleteEndpointOverride,
  getEndpointId,
} from '../utils/storage.js';
import type { OverridesStore, EndpointOverride, CustomParameter } from '../types/index.js';

interface UseOverridesReturn {
  // Get override for an endpoint
  getOverride: (method: string, path: string) => EndpointOverride | null;
  // Save override for an endpoint
  saveOverride: (
    method: string,
    path: string,
    params: Record<string, string>,
    customParams?: CustomParameter[],
    disabledParams?: string[],
    body?: string,
    overridePath?: string,
    overrideMethod?: string
  ) => Promise<void>;
  // Delete override for an endpoint
  deleteOverride: (method: string, path: string) => Promise<boolean>;
  // Check if endpoint has override
  hasOverride: (method: string, path: string) => boolean;
  // Check if endpoint has path/method override
  hasPathMethodOverride: (method: string, path: string) => boolean;
  // Check if endpoint has body override
  hasBodyOverride: (method: string, path: string) => boolean;
  // Check if endpoint has parameter overrides
  hasParamsOverride: (method: string, path: string) => boolean;
  // Reload overrides from disk
  reload: () => Promise<void>;
  // Loading state
  loading: boolean;
}

export function useOverrides(): UseOverridesReturn {
  const [store, setStore] = useState<OverridesStore>({ version: '1.0', endpoints: {} });
  const [loading, setLoading] = useState(true);

  // Load overrides on mount
  useEffect(() => {
    loadOverrides().then(data => {
      setStore(data);
      setLoading(false);
    });
  }, []);

  const reload = useCallback(async () => {
    setLoading(true);
    const data = await loadOverrides();
    setStore(data);
    setLoading(false);
  }, []);

  const getOverride = useCallback((method: string, path: string): EndpointOverride | null => {
    const id = getEndpointId(method, path);
    const override = store.endpoints[id];
    if (!override) return null;
    // Ensure all fields exist (for backwards compatibility)
    return {
      params: override.params || {},
      customParams: override.customParams || [],
      disabledParams: override.disabledParams || [],
      body: override.body,
      overridePath: override.overridePath,
      overrideMethod: override.overrideMethod,
      lastUsed: override.lastUsed,
    };
  }, [store]);

  const saveOverride = useCallback(async (
    method: string,
    path: string,
    params: Record<string, string>,
    customParams: CustomParameter[] = [],
    disabledParams: string[] = [],
    body?: string,
    overridePath?: string,
    overrideMethod?: string
  ): Promise<void> => {
    await saveEndpointOverride(method, path, params, customParams, disabledParams, body, overridePath, overrideMethod);
    // Update local state
    const id = getEndpointId(method, path);
    setStore(prev => ({
      ...prev,
      endpoints: {
        ...prev.endpoints,
        [id]: {
          params,
          customParams,
          disabledParams,
          body,
          overridePath,
          overrideMethod,
          lastUsed: new Date().toISOString(),
        },
      },
    }));
  }, []);

  const deleteOverride = useCallback(async (method: string, path: string): Promise<boolean> => {
    const result = await deleteEndpointOverride(method, path);
    if (result) {
      const id = getEndpointId(method, path);
      setStore(prev => {
        const newEndpoints = { ...prev.endpoints };
        delete newEndpoints[id];
        return { ...prev, endpoints: newEndpoints };
      });
    }
    return result;
  }, []);

  const hasOverride = useCallback((method: string, path: string): boolean => {
    const id = getEndpointId(method, path);
    return !!store.endpoints[id];
  }, [store]);

  const hasPathMethodOverride = useCallback((method: string, path: string): boolean => {
    const id = getEndpointId(method, path);
    const override = store.endpoints[id];
    if (!override) return false;
    return !!(override.overridePath || override.overrideMethod);
  }, [store]);

  const hasBodyOverride = useCallback((method: string, path: string): boolean => {
    const id = getEndpointId(method, path);
    const override = store.endpoints[id];
    return !!(override?.body);
  }, [store]);

  const hasParamsOverride = useCallback((method: string, path: string): boolean => {
    const id = getEndpointId(method, path);
    const override = store.endpoints[id];
    if (!override) return false;
    return Object.values(override.params || {}).some(v => v !== '')
      || (override.customParams?.length ?? 0) > 0
      || (override.disabledParams?.length ?? 0) > 0;
  }, [store]);

  return {
    getOverride,
    saveOverride,
    deleteOverride,
    hasOverride,
    hasPathMethodOverride,
    hasBodyOverride,
    hasParamsOverride,
    reload,
    loading,
  };
}

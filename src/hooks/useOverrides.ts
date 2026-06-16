import { useCallback } from 'react';
import { loadOverrides, saveOverridesStore, getEndpointId } from '../utils/persistence.js';
import { useStorage } from './useStorage.js';
import type { OverridesStore, EndpointOverride, CustomParameter } from '../types/index.js';

const INITIAL: OverridesStore = { version: '1.0', endpoints: {} };

interface UseOverridesReturn {
  getOverride: (method: string, path: string) => EndpointOverride | null;
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
  deleteOverride: (method: string, path: string) => Promise<boolean>;
  hasOverride: (method: string, path: string) => boolean;
  hasPathMethodOverride: (method: string, path: string) => boolean;
  hasBodyOverride: (method: string, path: string) => boolean;
  hasParamsOverride: (method: string, path: string) => boolean;
  loading: boolean;
}

export function useOverrides(): UseOverridesReturn {
  const [store, setStore, loading] = useStorage(loadOverrides, saveOverridesStore, INITIAL);

  const getOverride = useCallback((method: string, path: string): EndpointOverride | null => {
    const id = getEndpointId(method, path);
    const override = store.endpoints[id];
    if (!override) return null;
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
    const id = getEndpointId(method, path);
    setStore(prev => ({
      ...prev,
      endpoints: {
        ...prev.endpoints,
        [id]: { params, customParams, disabledParams, body, overridePath, overrideMethod, lastUsed: new Date().toISOString() },
      },
    }));
  }, [setStore]);

  const deleteOverride = useCallback(async (method: string, path: string): Promise<boolean> => {
    const id = getEndpointId(method, path);
    if (!store.endpoints[id]) return false;
    setStore(prev => {
      const endpoints = { ...prev.endpoints };
      delete endpoints[id];
      return { ...prev, endpoints };
    });
    return true;
  }, [store, setStore]);

  const hasOverride = useCallback((method: string, path: string): boolean => {
    return !!store.endpoints[getEndpointId(method, path)];
  }, [store]);

  const hasPathMethodOverride = useCallback((method: string, path: string): boolean => {
    const override = store.endpoints[getEndpointId(method, path)];
    return !!(override?.overridePath || override?.overrideMethod);
  }, [store]);

  const hasBodyOverride = useCallback((method: string, path: string): boolean => {
    return !!store.endpoints[getEndpointId(method, path)]?.body;
  }, [store]);

  const hasParamsOverride = useCallback((method: string, path: string): boolean => {
    const override = store.endpoints[getEndpointId(method, path)];
    if (!override) return false;
    return Object.values(override.params || {}).some(v => v !== '')
      || (override.customParams?.length ?? 0) > 0
      || (override.disabledParams?.length ?? 0) > 0;
  }, [store]);

  return { getOverride, saveOverride, deleteOverride, hasOverride, hasPathMethodOverride, hasBodyOverride, hasParamsOverride, loading };
}

import { useCallback } from 'react';
import { loadAuth, saveAuth } from '../utils/storage.js';
import { useStorage } from './useStorage.js';
import type { AuthStore } from '../types/index.js';

const INITIAL: AuthStore = { version: '1.0', credentials: {} };

export function useAuth() {
  const [store, setStore] = useStorage(loadAuth, saveAuth, INITIAL);

  const setCredential = useCallback((schemeName: string, value: string) => {
    setStore({ ...store, credentials: { ...store.credentials, [schemeName]: value } });
  }, [store, setStore]);

  const getCredential = useCallback((schemeName: string): string => {
    return store.credentials[schemeName] ?? '';
  }, [store]);

  return { store, setCredential, getCredential };
}

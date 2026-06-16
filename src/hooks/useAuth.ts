import { useState, useCallback, useEffect } from 'react';
import { loadAuth, saveAuth } from '../utils/storage.js';
import type { AuthStore } from '../types/index.js';

export function useAuth() {
  const [store, setStore] = useState<AuthStore>({ version: '1.0', credentials: {} });

  useEffect(() => {
    loadAuth().then(setStore);
  }, []);

  const setCredential = useCallback((schemeName: string, value: string) => {
    setStore(prev => {
      const next = { ...prev, credentials: { ...prev.credentials, [schemeName]: value } };
      saveAuth(next);
      return next;
    });
  }, []);

  const getCredential = useCallback((schemeName: string): string => {
    return store.credentials[schemeName] || '';
  }, [store]);

  return { store, setCredential, getCredential };
}

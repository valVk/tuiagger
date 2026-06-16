import { useState, useEffect, useCallback } from 'react';
import {
  loadSavedRequests,
  addSavedRequest,
  updateSavedRequest,
  deleteSavedRequest,
  addCustomTag,
} from '../utils/storage.js';
import type { SavedRequest, SavedRequestsStore, CustomTag } from '../types/index.js';

interface UseSavedRequestsResult {
  requests: SavedRequest[];
  customTags: CustomTag[];
  loading: boolean;
  save: (request: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'>) => Promise<SavedRequest>;
  update: (id: string, updates: Partial<SavedRequest>) => Promise<SavedRequest | null>;
  remove: (id: string) => Promise<boolean>;
  getRequestsByTag: (tag: string) => SavedRequest[];
  getAllTags: (specTags: string[]) => string[];
  createTag: (tag: CustomTag) => Promise<void>;
}

export function useSavedRequests(): UseSavedRequestsResult {
  const [store, setStore] = useState<SavedRequestsStore>({
    version: '1.0',
    requests: [],
    customTags: [],
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadSavedRequests().then(data => {
      setStore(data);
      setLoading(false);
    });
  }, []);

  const save = useCallback(
    async (request: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'>) => {
      const saved = await addSavedRequest(request);
      setStore(prev => ({
        ...prev,
        requests: [...prev.requests, saved],
      }));
      return saved;
    },
    []
  );

  const update = useCallback(async (id: string, updates: Partial<SavedRequest>) => {
    const updated = await updateSavedRequest(id, updates);
    if (updated) {
      setStore(prev => ({
        ...prev,
        requests: prev.requests.map(r => (r.id === id ? updated : r)),
      }));
    }
    return updated;
  }, []);

  const remove = useCallback(async (id: string) => {
    const success = await deleteSavedRequest(id);
    if (success) {
      setStore(prev => ({
        ...prev,
        requests: prev.requests.filter(r => r.id !== id),
      }));
    }
    return success;
  }, []);

  const getRequestsByTag = useCallback(
    (tag: string) => {
      return store.requests.filter(r => r.tag === tag);
    },
    [store.requests]
  );

  const getAllTags = useCallback(
    (specTags: string[]) => {
      const customTagNames = store.customTags.map(t => t.name);
      return [...new Set([...specTags, ...customTagNames])];
    },
    [store.customTags]
  );

  const createTag = useCallback(async (tag: CustomTag) => {
    await addCustomTag(tag);
    setStore(prev => ({
      ...prev,
      customTags: [...prev.customTags, tag],
    }));
  }, []);

  return {
    requests: store.requests,
    customTags: store.customTags,
    loading,
    save,
    update,
    remove,
    getRequestsByTag,
    getAllTags,
    createTag,
  };
}

import { useCallback } from 'react';
import { randomUUID } from 'crypto';
import { loadSavedRequests, saveSavedRequests } from '../utils/persistence.js';
import { useStorage } from './useStorage.js';
import type { SavedRequest, SavedRequestsStore, CustomTag } from '../types/index.js';

const INITIAL: SavedRequestsStore = { version: '1.0', requests: [], customTags: [] };

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
  renameTag: (oldName: string, newName: string) => Promise<boolean>;
  deleteTag: (name: string) => Promise<void>;
}

export function useSavedRequests(): UseSavedRequestsResult {
  const [store, setStore, loading] = useStorage(loadSavedRequests, saveSavedRequests, INITIAL);

  const save = useCallback(async (request: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'>): Promise<SavedRequest> => {
    const now = new Date().toISOString();
    const newRequest: SavedRequest = { ...request, id: randomUUID(), createdAt: now, updatedAt: now };
    setStore(prev => ({ ...prev, requests: [...prev.requests, newRequest] }));
    return newRequest;
  }, [setStore]);

  const update = useCallback(async (id: string, updates: Partial<SavedRequest>): Promise<SavedRequest | null> => {
    const existing = store.requests.find(r => r.id === id);
    if (!existing) return null;
    const updated: SavedRequest = { ...existing, ...updates, updatedAt: new Date().toISOString() };
    setStore(prev => ({ ...prev, requests: prev.requests.map(r => r.id === id ? updated : r) }));
    return updated;
  }, [store, setStore]);

  const remove = useCallback(async (id: string): Promise<boolean> => {
    if (!store.requests.find(r => r.id === id)) return false;
    setStore(prev => ({ ...prev, requests: prev.requests.filter(r => r.id !== id) }));
    return true;
  }, [store, setStore]);

  const getRequestsByTag = useCallback((tag: string) => {
    return store.requests.filter(r => r.tag === tag);
  }, [store.requests]);

  const getAllTags = useCallback((specTags: string[]) => {
    const customTagNames = store.customTags.map(t => t.name);
    const requestTagNames = store.requests.map(r => r.tag);
    return [...new Set([...specTags, ...customTagNames, ...requestTagNames])];
  }, [store.customTags, store.requests]);

  const createTag = useCallback(async (tag: CustomTag): Promise<void> => {
    if (store.customTags.find(t => t.name === tag.name)) return;
    setStore(prev => ({ ...prev, customTags: [...prev.customTags, tag] }));
  }, [store, setStore]);

  const renameTag = useCallback(async (oldName: string, newName: string): Promise<boolean> => {
    if (!newName || oldName === newName) return false;
    if (store.customTags.some(t => t.name === newName) || store.requests.some(r => r.tag === newName)) return false;
    const now = new Date().toISOString();
    setStore(prev => ({
      ...prev,
      customTags: prev.customTags.map(t => t.name === oldName ? { ...t, name: newName } : t),
      requests: prev.requests.map(r => r.tag === oldName ? { ...r, tag: newName, updatedAt: now } : r),
    }));
    return true;
  }, [store, setStore]);

  const deleteTag = useCallback(async (name: string): Promise<void> => {
    setStore(prev => ({
      ...prev,
      customTags: prev.customTags.filter(t => t.name !== name),
      requests: prev.requests.filter(r => r.tag !== name),
    }));
  }, [setStore]);

  return { requests: store.requests, customTags: store.customTags, loading, save, update, remove, getRequestsByTag, getAllTags, createTag, renameTag, deleteTag };
}

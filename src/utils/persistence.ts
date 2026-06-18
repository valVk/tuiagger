import { readFile, writeFile, mkdir, rename } from 'fs/promises';
import { join, dirname } from 'path';
import { randomUUID } from 'crypto';
import { getCollectionPath } from './collectionResolver.js';
import type { SavedRequest, SavedRequestsStore, CustomTag, OverridesStore, EndpointOverride, CustomParameter, AuthStore, EnvironmentsStore } from '../types/index.js';

const STORAGE_DIR = '.tuiagger';
const STORAGE_FILE = 'saved-requests.json';
const OVERRIDES_FILE = 'overrides.json';
const AUTH_FILE = 'auth.json';
const ENVIRONMENTS_FILE = 'environments.json';

function getSavedRequestsPath(): string {
  return join(process.cwd(), STORAGE_DIR, STORAGE_FILE);
}

function getOverridesPath(): string {
  const col = getCollectionPath();
  if (col) return join(col, OVERRIDES_FILE);
  return join(process.cwd(), STORAGE_DIR, OVERRIDES_FILE);
}

function getAuthPath(): string {
  const col = getCollectionPath();
  if (col) return join(col, AUTH_FILE);
  return join(process.cwd(), STORAGE_DIR, AUTH_FILE);
}

function getEnvironmentsPath(): string {
  const col = getCollectionPath();
  if (col) return join(col, ENVIRONMENTS_FILE);
  return join(process.cwd(), STORAGE_DIR, ENVIRONMENTS_FILE);
}

async function atomicWrite(path: string, data: string): Promise<void> {
  const tempPath = path + '.tmp';
  await mkdir(dirname(path), { recursive: true });
  await writeFile(tempPath, data);
  await rename(tempPath, path);
}

// ============ Saved Requests ============

export async function loadSavedRequests(): Promise<SavedRequestsStore> {
  try {
    const data = await readFile(getSavedRequestsPath(), 'utf-8');
    return JSON.parse(data) as SavedRequestsStore;
  } catch {
    return { version: '1.0', requests: [], customTags: [] };
  }
}

export async function saveSavedRequests(store: SavedRequestsStore): Promise<void> {
  const path = getSavedRequestsPath();
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, JSON.stringify(store, null, 2));
}

export async function addSavedRequest(
  request: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'>
): Promise<SavedRequest> {
  const store = await loadSavedRequests();
  const newRequest: SavedRequest = {
    ...request,
    id: randomUUID(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
  store.requests.push(newRequest);
  await saveSavedRequests(store);
  return newRequest;
}

export async function updateSavedRequest(
  id: string,
  updates: Partial<Omit<SavedRequest, 'id' | 'createdAt'>>
): Promise<SavedRequest | null> {
  const store = await loadSavedRequests();
  const index = store.requests.findIndex(r => r.id === id);
  if (index === -1) return null;
  store.requests[index] = { ...store.requests[index], ...updates, updatedAt: new Date().toISOString() };
  await saveSavedRequests(store);
  return store.requests[index];
}

export async function deleteSavedRequest(id: string): Promise<boolean> {
  const store = await loadSavedRequests();
  const initialLength = store.requests.length;
  store.requests = store.requests.filter(r => r.id !== id);
  if (store.requests.length < initialLength) {
    await saveSavedRequests(store);
    return true;
  }
  return false;
}

export async function addCustomTag(tag: CustomTag): Promise<void> {
  const store = await loadSavedRequests();
  if (!store.customTags.find(t => t.name === tag.name)) {
    store.customTags.push(tag);
    await saveSavedRequests(store);
  }
}

export async function deleteCustomTag(tagName: string): Promise<boolean> {
  const store = await loadSavedRequests();
  const initialLength = store.customTags.length;
  store.customTags = store.customTags.filter(t => t.name !== tagName);
  if (store.customTags.length < initialLength) {
    await saveSavedRequests(store);
    return true;
  }
  return false;
}

// ============ Overrides ============

export function getEndpointId(method: string, path: string): string {
  return `${method.toUpperCase()} ${path}`;
}

export async function loadOverrides(): Promise<OverridesStore> {
  try {
    const data = await readFile(getOverridesPath(), 'utf-8');
    return JSON.parse(data) as OverridesStore;
  } catch {
    return { version: '1.0', endpoints: {} };
  }
}

export async function saveOverridesStore(store: OverridesStore): Promise<void> {
  await atomicWrite(getOverridesPath(), JSON.stringify(store, null, 2));
}

export async function getEndpointOverride(method: string, path: string): Promise<EndpointOverride | null> {
  const store = await loadOverrides();
  const id = getEndpointId(method, path);
  return store.endpoints[id] || null;
}

export async function saveEndpointOverride(
  method: string,
  path: string,
  params: Record<string, string>,
  customParams: CustomParameter[] = [],
  disabledParams: string[] = [],
  body?: string,
  overridePath?: string,
  overrideMethod?: string
): Promise<void> {
  const store = await loadOverrides();
  const id = getEndpointId(method, path);
  store.endpoints[id] = { params, customParams, disabledParams, body, overridePath, overrideMethod, lastUsed: new Date().toISOString() };
  await saveOverridesStore(store);
}

export async function deleteEndpointOverride(method: string, path: string): Promise<boolean> {
  const store = await loadOverrides();
  const id = getEndpointId(method, path);
  if (store.endpoints[id]) {
    delete store.endpoints[id];
    await saveOverridesStore(store);
    return true;
  }
  return false;
}

// ============ Auth ============

export async function loadAuth(): Promise<AuthStore> {
  try {
    const data = await readFile(getAuthPath(), 'utf-8');
    return JSON.parse(data) as AuthStore;
  } catch {
    return { version: '1.0', credentials: {} };
  }
}

export async function saveAuth(store: AuthStore): Promise<void> {
  await atomicWrite(getAuthPath(), JSON.stringify(store, null, 2));
}

// ============ Environments ============

export async function loadEnvironments(): Promise<EnvironmentsStore> {
  try {
    const data = await readFile(getEnvironmentsPath(), 'utf-8');
    return JSON.parse(data) as EnvironmentsStore;
  } catch {
    return { version: '1.0', environments: [], activeIndex: -1 };
  }
}

export async function saveEnvironments(store: EnvironmentsStore): Promise<void> {
  await atomicWrite(getEnvironmentsPath(), JSON.stringify(store, null, 2));
}

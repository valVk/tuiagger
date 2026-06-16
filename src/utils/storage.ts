import { readFile, writeFile, mkdir, readdir, access, rename } from 'fs/promises';
import { join, dirname } from 'path';
import { homedir } from 'os';
import { randomUUID } from 'crypto';
import type { SavedRequest, SavedRequestsStore, CustomTag, OverridesStore, EndpointOverride, CustomParameter, AuthStore, EnvironmentsStore } from '../types/index.js';

const STORAGE_DIR = '.twagger';
const STORAGE_FILE = 'saved-requests.json';
const OVERRIDES_FILE = 'overrides.json';

// Current collection path for per-collection storage
let currentCollectionPath: string | null = null;

export function setCollectionPath(path: string | null): void {
  currentCollectionPath = path;
}

export function getCollectionPath(): string | null {
  return currentCollectionPath;
}

function getBaseDir(): string {
  return join(homedir(), STORAGE_DIR);
}

function getSavedRequestsPath(): string {
  return join(process.cwd(), STORAGE_DIR, STORAGE_FILE);
}

export interface CollectionConfig {
  name: string;
  source: string;
  path: string;
}

export async function resolveCollection(nameOrPath: string): Promise<CollectionConfig | null> {
  // Check if it's a URL
  if (nameOrPath.startsWith('http://') || nameOrPath.startsWith('https://')) {
    return {
      name: 'Remote',
      source: nameOrPath,
      path: nameOrPath,
    };
  }

  // Check if it's a file path (contains path separators or file extension)
  if (nameOrPath.includes('/') || nameOrPath.includes('\\') || nameOrPath.endsWith('.json') || nameOrPath.endsWith('.yaml') || nameOrPath.endsWith('.yml')) {
    return {
      name: 'Local',
      source: nameOrPath,
      path: nameOrPath,
    };
  }

  // Treat as collection name - look in ~/.twagger/<name>/
  const collectionDir = join(getBaseDir(), nameOrPath);

  try {
    await access(collectionDir);
  } catch {
    return null;
  }

  // Find JSON/YAML file in the directory — exclude internal storage files
  const INTERNAL_FILES = new Set(['auth.json', 'overrides.json', 'saved-requests.json', 'environments.json']);
  const files = await readdir(collectionDir);
  const specFile = files.find(f =>
    (f.endsWith('.json') || f.endsWith('.yaml') || f.endsWith('.yml')) &&
    !INTERNAL_FILES.has(f)
  );

  if (!specFile) {
    return null;
  }

  return {
    name: nameOrPath,
    source: join(collectionDir, specFile),
    path: collectionDir,
  };
}

export async function listCollections(): Promise<string[]> {
  try {
    const baseDir = getBaseDir();
    const entries = await readdir(baseDir, { withFileTypes: true });
    return entries
      .filter(e => e.isDirectory())
      .map(e => e.name);
  } catch {
    return [];
  }
}

export async function loadSavedRequests(): Promise<SavedRequestsStore> {
  try {
    const path = getSavedRequestsPath();
    const data = await readFile(path, 'utf-8');
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

  store.requests[index] = {
    ...store.requests[index],
    ...updates,
    updatedAt: new Date().toISOString(),
  };
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

// ============ Overrides Storage ============

function getOverridesPath(): string {
  if (currentCollectionPath) {
    return join(currentCollectionPath, OVERRIDES_FILE);
  }
  // Fallback to cwd for non-collection usage
  return join(process.cwd(), STORAGE_DIR, OVERRIDES_FILE);
}

export async function loadOverrides(): Promise<OverridesStore> {
  try {
    const path = getOverridesPath();
    const data = await readFile(path, 'utf-8');
    return JSON.parse(data) as OverridesStore;
  } catch {
    return { version: '1.0', endpoints: {} };
  }
}

// Atomic write to prevent corruption
async function saveOverrides(store: OverridesStore): Promise<void> {
  const path = getOverridesPath();
  const tempPath = path + '.tmp';
  await mkdir(dirname(path), { recursive: true });
  await writeFile(tempPath, JSON.stringify(store, null, 2));
  await rename(tempPath, path);
}

export function getEndpointId(method: string, path: string): string {
  return `${method.toUpperCase()} ${path}`;
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

  store.endpoints[id] = {
    params,
    customParams,
    disabledParams,
    body,
    overridePath,
    overrideMethod,
    lastUsed: new Date().toISOString(),
  };

  await saveOverrides(store);
}

export async function deleteEndpointOverride(method: string, path: string): Promise<boolean> {
  const store = await loadOverrides();
  const id = getEndpointId(method, path);

  if (store.endpoints[id]) {
    delete store.endpoints[id];
    await saveOverrides(store);
    return true;
  }
  return false;
}

// ============ Auth Storage ============

const AUTH_FILE = 'auth.json';

function getAuthPath(): string {
  if (currentCollectionPath) return join(currentCollectionPath, AUTH_FILE);
  return join(process.cwd(), STORAGE_DIR, AUTH_FILE);
}

export async function loadAuth(): Promise<AuthStore> {
  try {
    const data = await readFile(getAuthPath(), 'utf-8');
    return JSON.parse(data) as AuthStore;
  } catch {
    return { version: '1.0', credentials: {} };
  }
}

export async function saveAuth(store: AuthStore): Promise<void> {
  const path = getAuthPath();
  const tempPath = path + '.tmp';
  await mkdir(dirname(path), { recursive: true });
  await writeFile(tempPath, JSON.stringify(store, null, 2));
  await rename(tempPath, path);
}

// ============ Environments Storage ============

const ENVIRONMENTS_FILE = 'environments.json';

function getEnvironmentsPath(): string {
  if (currentCollectionPath) return join(currentCollectionPath, ENVIRONMENTS_FILE);
  return join(process.cwd(), STORAGE_DIR, ENVIRONMENTS_FILE);
}

export async function loadEnvironments(): Promise<EnvironmentsStore> {
  try {
    const data = await readFile(getEnvironmentsPath(), 'utf-8');
    return JSON.parse(data) as EnvironmentsStore;
  } catch {
    return { version: '1.0', environments: [], activeIndex: -1 };
  }
}

export async function saveEnvironments(store: EnvironmentsStore): Promise<void> {
  const path = getEnvironmentsPath();
  const tempPath = path + '.tmp';
  await mkdir(dirname(path), { recursive: true });
  await writeFile(tempPath, JSON.stringify(store, null, 2));
  await rename(tempPath, path);
}

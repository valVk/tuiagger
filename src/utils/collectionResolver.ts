import { readdir, access } from 'fs/promises';
import { join } from 'path';
import { homedir } from 'os';

const STORAGE_DIR = '.twagger';

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

export interface CollectionConfig {
  name: string;
  source: string;
  path: string;
}

export async function resolveCollection(nameOrPath: string): Promise<CollectionConfig | null> {
  if (nameOrPath.startsWith('http://') || nameOrPath.startsWith('https://')) {
    return { name: 'Remote', source: nameOrPath, path: nameOrPath };
  }

  if (nameOrPath.includes('/') || nameOrPath.includes('\\') || nameOrPath.endsWith('.json') || nameOrPath.endsWith('.yaml') || nameOrPath.endsWith('.yml')) {
    return { name: 'Local', source: nameOrPath, path: nameOrPath };
  }

  const collectionDir = join(getBaseDir(), nameOrPath);

  try {
    await access(collectionDir);
  } catch {
    return null;
  }

  const INTERNAL_FILES = new Set(['auth.json', 'overrides.json', 'saved-requests.json', 'environments.json']);
  const files = await readdir(collectionDir);
  const specFile = files.find(f =>
    (f.endsWith('.json') || f.endsWith('.yaml') || f.endsWith('.yml')) &&
    !INTERNAL_FILES.has(f)
  );

  if (!specFile) return null;

  return {
    name: nameOrPath,
    source: join(collectionDir, specFile),
    path: collectionDir,
  };
}

export async function listCollections(): Promise<string[]> {
  try {
    const entries = await readdir(getBaseDir(), { withFileTypes: true });
    return entries.filter(e => e.isDirectory()).map(e => e.name);
  } catch {
    return [];
  }
}

import { createInterface } from 'readline/promises';
import { mkdir, writeFile, access } from 'fs/promises';
import { join } from 'path';
import { homedir } from 'os';

const STORAGE_DIR = '.tuiagger';

async function prompt(rl: ReturnType<typeof createInterface>, question: string, defaultValue?: string): Promise<string> {
  const suffix = defaultValue ? ` (${defaultValue})` : '';
  const answer = (await rl.question(`${question}${suffix}: `)).trim();
  return answer || defaultValue || '';
}

async function promptServers(rl: ReturnType<typeof createInterface>): Promise<string[]> {
  const servers: string[] = [];
  console.log('Server URLs (blank line to finish):');
  while (true) {
    const url = (await rl.question(`  server[${servers.length}]: `)).trim();
    if (!url) break;
    servers.push(url);
  }
  return servers;
}

export async function runInit(name: string): Promise<void> {
  if (!name) {
    console.error('Error: Please provide a collection name\n');
    console.error('Usage: tuiagger init <name>');
    process.exit(1);
  }

  const collectionDir = join(homedir(), STORAGE_DIR, name);

  try {
    await access(collectionDir);
    console.error(`Error: Collection "${name}" already exists at ${collectionDir}`);
    process.exit(1);
  } catch {
    // does not exist yet, proceed
  }

  const rl = createInterface({ input: process.stdin, output: process.stdout });

  console.log(`This utility will walk you through creating an OpenAPI spec for "${name}".\n`);

  try {
    const title = await prompt(rl, 'title', name);
    const version = await prompt(rl, 'version', '1.0.0');
    const servers = await promptServers(rl);

    const spec: Record<string, unknown> = {
      openapi: '3.0.0',
      info: { title, version },
      paths: {},
    };

    if (servers.length > 0) {
      spec.servers = servers.map(url => ({ url }));
    }

    await mkdir(collectionDir, { recursive: true });
    await writeFile(join(collectionDir, 'openapi.json'), JSON.stringify(spec, null, 2));

    console.log(`\nCreated ${join(collectionDir, 'openapi.json')}`);
    console.log(`\nRun it with:\n  tuiagger ${name}`);
  } finally {
    rl.close();
  }
}

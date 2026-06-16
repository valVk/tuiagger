import { spawnSync } from 'child_process';
import { writeFileSync, readFileSync, unlinkSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';

export function openInEditor(content: string, extension = 'json'): string {
  const tmpFile = join(tmpdir(), `twagger-body-${Date.now()}.${extension}`);
  try {
    writeFileSync(tmpFile, content, 'utf-8');
    const editor = process.env.EDITOR || process.env.VISUAL || 'vi';
    spawnSync(editor, [tmpFile], { stdio: 'inherit' });
    return readFileSync(tmpFile, 'utf-8');
  } finally {
    try { unlinkSync(tmpFile); } catch {}
  }
}

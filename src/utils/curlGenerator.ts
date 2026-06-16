export function generateCurl(
  method: string,
  url: string,
  headers: Record<string, string>,
  body?: string
): string {
  const parts: string[] = [`curl -X '${method.toUpperCase()}'`];
  parts.push(`  '${url}'`);

  for (const [name, value] of Object.entries(headers)) {
    parts.push(`  -H '${name}: ${value}'`);
  }

  if (body) {
    // Escape single quotes in body
    const escapedBody = body.replace(/'/g, "'\\''");
    parts.push(`  -d '${escapedBody}'`);
  }

  return parts.join(' \\\n');
}

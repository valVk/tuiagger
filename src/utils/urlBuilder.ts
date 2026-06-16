import type { KeyValuePair } from '../types/index.js';

export function buildRequestUrl(
  baseUrl: string,
  path: string,
  pathParams: Record<string, string>,
  queryParams: Record<string, string>
): string {
  let url = path;

  // Replace path parameters
  for (const [name, value] of Object.entries(pathParams)) {
    url = url.replace(`{${name}}`, encodeURIComponent(value));
  }

  // Build full URL
  const fullUrl = new URL(url, baseUrl);

  // Add query parameters
  for (const [name, value] of Object.entries(queryParams)) {
    if (value) {
      fullUrl.searchParams.append(name, value);
    }
  }

  return fullUrl.toString();
}

export function buildManualRequestUrl(
  baseUrl: string,
  path: string,
  queryParams: KeyValuePair[]
): string {
  // Ensure baseUrl has a trailing slash for correct relative path resolution
  const baseWithSlash = baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`;

  // Ensure path is relative (doesn't start with a /)
  const relativePath = path.startsWith('/') ? path.slice(1) : path;

  // Build full URL
  const fullUrl = new URL(relativePath, baseWithSlash);

  // Add enabled query parameters with non-empty keys
  for (const param of queryParams) {
    if (param.enabled && param.key) {
      fullUrl.searchParams.append(param.key, param.value);
    }
  }

  return fullUrl.toString();
}

export function extractPathParams(path: string): string[] {
  const matches = path.match(/\{([^}]+)\}/g);
  if (!matches) return [];
  return matches.map(match => match.slice(1, -1));
}

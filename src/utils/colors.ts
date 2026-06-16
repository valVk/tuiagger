import type { HttpMethod } from '../types/index.js';

export type MethodColor = 'blue' | 'green' | 'yellow' | 'red' | 'cyan' | 'magenta' | 'gray';

export const METHOD_COLORS: Record<HttpMethod, MethodColor> = {
  get: 'blue',
  post: 'green',
  put: 'yellow',
  delete: 'red',
  patch: 'cyan',
  options: 'gray',
  head: 'magenta',
  trace: 'gray',
};

export const STATUS_COLORS: Record<string, string> = {
  '2xx': 'green',
  '3xx': 'cyan',
  '4xx': 'yellow',
  '5xx': 'red',
};

export function getStatusColor(status: number): string {
  if (status >= 200 && status < 300) return STATUS_COLORS['2xx'];
  if (status >= 300 && status < 400) return STATUS_COLORS['3xx'];
  if (status >= 400 && status < 500) return STATUS_COLORS['4xx'];
  if (status >= 500) return STATUS_COLORS['5xx'];
  return 'gray';
}

export function getMethodColor(method: string): MethodColor {
  const lowerMethod = method.toLowerCase() as HttpMethod;
  return METHOD_COLORS[lowerMethod] || 'gray';
}

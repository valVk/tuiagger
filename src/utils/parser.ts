import { parse } from '@readme/openapi-parser';
import { Parser } from 'htmlparser2';
import type {
  OpenAPISpec,
  PathItemObject,
  OperationObject,
  HttpMethod,
} from '../types/index.js';

export interface ParsedEndpoint {
  path: string;
  method: HttpMethod;
  operation: OperationObject;
  tags: string[];
}

export interface ParsedSpec {
  spec: OpenAPISpec;
  endpoints: ParsedEndpoint[];
  tags: string[];
}

export async function parseOpenAPISpec(source: string): Promise<ParsedSpec> {
  let spec: OpenAPISpec;

  // The parse function handles both URLs and local file paths directly.
  spec = (await parse(source)) as OpenAPISpec;

  const endpoints = extractEndpoints(spec);
  const tags = extractTags(spec, endpoints);

  return { spec, endpoints, tags };
}

function extractEndpoints(spec: OpenAPISpec): ParsedEndpoint[] {
  const endpoints: ParsedEndpoint[] = [];
  const methods: HttpMethod[] = ['get', 'post', 'put', 'delete', 'patch', 'options', 'head', 'trace'];

  for (const [path, pathItem] of Object.entries(spec.paths)) {
    for (const method of methods) {
      const operation = pathItem[method as keyof PathItemObject] as OperationObject | undefined;
      if (operation) {
        endpoints.push({
          path,
          method,
          operation,
          tags: operation.tags || ['default'],
        });
      }
    }
  }

  return endpoints;
}

function extractTags(spec: OpenAPISpec, endpoints: ParsedEndpoint[]): string[] {
  // Get tags from spec definition
  const specTags = spec.tags?.map(t => t.name) || [];

  // Get tags from endpoints that might not be in spec definition
  const endpointTags = new Set<string>();
  for (const endpoint of endpoints) {
    for (const tag of endpoint.tags) {
      endpointTags.add(tag);
    }
  }

  // Merge and deduplicate, preserving spec order
  const allTags = [...specTags];
  for (const tag of endpointTags) {
    if (!allTags.includes(tag)) {
      allTags.push(tag);
    }
  }

  return allTags;
}

export function getEndpointsByTag(endpoints: ParsedEndpoint[]): Map<string, ParsedEndpoint[]> {
  const byTag = new Map<string, ParsedEndpoint[]>();

  for (const endpoint of endpoints) {
    for (const tag of endpoint.tags) {
      const existing = byTag.get(tag) || [];
      existing.push(endpoint);
      byTag.set(tag, existing);
    }
  }

  return byTag;
}

export function htmlToPlainText(html: string): string {
  let textContent = '';

  const parser = new Parser({
    ontext(text) {
      textContent += text;
    },
    onclosetag(name) {
      textContent += '\n';
    },
  }, { decodeEntities: true });

  parser.write(html);
  parser.end();

  return textContent.replace(/\n\s*\n/g, '\n').trim(); // Normalize multiple newlines and trim
}

// Resolves a JSON Pointer $ref string (e.g. "#/components/schemas/Pet") against components.
export function resolveRef(ref: string, components: Record<string, unknown>): unknown {
  const parts = ref.replace('#/components/', '').split('/');
  let resolved: unknown = components;
  for (const part of parts) {
    resolved = (resolved as Record<string, unknown>)?.[part];
  }
  return resolved ?? null;
}

// Resolves $ref and guards against circular references.
// Returns [resolvedSchema, shouldSkip] — shouldSkip true means caller should bail out.
export function resolveSchema(
  schema: unknown,
  components: Record<string, unknown> | undefined,
  seen: Set<unknown>
): [Record<string, unknown> | null, boolean] {
  if (!schema || typeof schema !== 'object') return [null, true];
  if (seen.has(schema)) return [null, true];

  const s = schema as Record<string, unknown>;

  if (s.$ref && typeof s.$ref === 'string') {
    if (!components) return [null, true];
    const resolved = resolveRef(s.$ref, components);
    if (!resolved || typeof resolved !== 'object') return [null, true];
    seen.add(schema);
    return resolveSchema(resolved, components, seen);
  }

  seen.add(s);
  return [s, false];
}

export function formatSchema(schema: unknown, indent = 0, components?: Record<string, unknown>, seen = new Set<unknown>()): string {
  const [s, skip] = resolveSchema(schema, components, seen);
  if (skip || !s) {
    if (!schema || typeof schema !== 'object') return String(schema);
    return '(circular)';
  }

  const spaces = '  '.repeat(indent);
  const lines: string[] = [];

  if (s.type === 'object' && s.properties) {
    lines.push(`${spaces}{`);
    const props = s.properties as Record<string, unknown>;
    const required = (s.required as string[]) || [];
    for (const [key, value] of Object.entries(props)) {
      const isRequired = required.includes(key);
      const propSchema = formatSchema(value, indent + 1, components, seen);
      lines.push(`${spaces}  "${key}"${isRequired ? ' *' : ''}: ${propSchema}`);
    }
    lines.push(`${spaces}}`);
  } else if (s.type === 'array' && s.items) {
    const itemsSchema = formatSchema(s.items, indent, components, seen);
    lines.push(`${spaces}[${itemsSchema}]`);
  } else if (s.type) {
    let typeStr = s.type as string;
    if (s.format) typeStr += `(${s.format})`;
    if (s.enum) typeStr += ` enum: [${(s.enum as unknown[]).join(', ')}]`;
    return typeStr;
  }

  return lines.join('\n') || JSON.stringify(s, null, 2);
}

export function scaffoldPlaceholder(schema: unknown, components?: Record<string, unknown>, seen = new Set<unknown>()): unknown {
  const [s, skip] = resolveSchema(schema, components, seen);
  if (skip || !s) return null;

  if (s.type === 'object' && s.properties) {
    const result: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(s.properties as Record<string, unknown>)) {
      result[key] = scaffoldPlaceholder(value, components, seen);
    }
    return result;
  }

  if (s.type === 'array' && s.items) {
    return [scaffoldPlaceholder(s.items, components, seen)];
  }

  if (s.enum) return (s.enum as unknown[]).join(' | ');
  if (s.type === 'string') return s.format ? `<${s.format}>` : '<string>';
  if (s.type === 'integer' || s.type === 'number') return 0;
  if (s.type === 'boolean') return false;

  return null;
}

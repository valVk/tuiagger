import { faker } from '@faker-js/faker';
import { resolveSchema } from './parser.js';

export function scaffoldBody(
  schema: Record<string, unknown>,
  components?: Record<string, unknown>,
  seen = new Set<unknown>()
): unknown {
  const [resolved, skip] = resolveSchema(schema, components, seen);
  if (skip || !resolved) return null;

  if (resolved.example !== undefined) return resolved.example;

  if (Array.isArray(resolved.enum) && resolved.enum.length > 0) {
    return faker.helpers.arrayElement(resolved.enum as unknown[]);
  }

  const type = resolved.type as string | undefined;
  const format = resolved.format as string | undefined;

  if (type === 'string') return scaffoldString(format);
  if (type === 'integer') return scaffoldInteger(format, resolved);
  if (type === 'number') return scaffoldNumber(format);
  if (type === 'boolean') return faker.datatype.boolean();

  if (type === 'array') {
    const items = resolved.items as Record<string, unknown> | undefined;
    if (!items) return [];
    const item = scaffoldBody(items, components, new Set(seen));
    return item !== null ? [item] : [];
  }

  if (type === 'object' || resolved.properties) {
    const props = resolved.properties as Record<string, Record<string, unknown>> | undefined;
    if (!props) return {};
    const result: Record<string, unknown> = {};
    for (const [key, propSchema] of Object.entries(props)) {
      const val = scaffoldBody(propSchema, components, new Set(seen));
      if (val !== null) result[key] = val;
    }
    return result;
  }

  for (const key of ['allOf', 'oneOf', 'anyOf'] as const) {
    const list = resolved[key] as Record<string, unknown>[] | undefined;
    if (list?.length) {
      const val = scaffoldBody(list[0], components, new Set(seen));
      if (val !== null) return val;
    }
  }

  return null;
}

function scaffoldString(format?: string): string {
  switch (format) {
    case 'uuid':      return faker.string.uuid();
    case 'email':     return faker.internet.email();
    case 'date':      return faker.date.past().toISOString().split('T')[0];
    case 'date-time': return faker.date.past().toISOString();
    case 'uri':
    case 'url':       return faker.internet.url();
    case 'hostname':  return faker.internet.domainName();
    case 'ipv4':      return faker.internet.ipv4();
    case 'ipv6':      return faker.internet.ipv6();
    case 'byte':      return Buffer.from(faker.lorem.word()).toString('base64');
    case 'password':  return faker.internet.password();
    default:          return faker.lorem.word();
  }
}

function scaffoldInteger(format?: string, schema?: Record<string, unknown>): number {
  const min = schema?.minimum as number | undefined;
  const max = schema?.maximum as number | undefined;
  if (format === 'int32') {
    return faker.number.int({ min: min ?? -2147483648, max: max ?? 2147483647 });
  }
  return faker.number.int({ min, max });
}

function scaffoldNumber(format?: string): number {
  if (format === 'float') return faker.number.float({ fractionDigits: 2 });
  return faker.number.float({ fractionDigits: 4 });
}

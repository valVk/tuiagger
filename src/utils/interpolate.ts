import { faker } from '@faker-js/faker';

const INTERPOLATION_RE = /\{\{faker\.([a-zA-Z]+)\.([a-zA-Z]+)\(\)\}\}/g;
const ENV_VAR_RE = /\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}/g;

export function interpolate(value: string, envVars?: Record<string, string>): string {
  // Faker expressions first
  let result = value.replace(INTERPOLATION_RE, (_, module, method) => {
    try {
      const mod = (faker as unknown as Record<string, unknown>)[module];
      if (mod && typeof (mod as Record<string, unknown>)[method] === 'function') {
        return String((mod as Record<string, Function>)[method]());
      }
    } catch {
      // unknown faker path — leave expression intact
    }
    return `{{faker.${module}.${method}()}}`;
  });

  // Then env variable substitution — multi-pass to resolve chains (max 10 iterations)
  if (envVars) {
    for (let i = 0; i < 10; i++) {
      const next = result.replace(ENV_VAR_RE, (_, key) => envVars[key] ?? `{{${key}}}`);
      if (next === result) break;
      result = next;
    }
  }

  return result;
}

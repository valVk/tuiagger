import { interpolate } from './interpolate.js';
import type { KeyValuePair, CustomParameter } from '../types/index.js';
import type { ParameterObject } from '../types/openapi.js';

export class ParameterCollector {
  constructor(
    private readonly specParams: ParameterObject[],
    private readonly customParams: CustomParameter[],
    private readonly disabledParams: string[],
    private readonly parameterValues: Record<string, string>,
    private readonly envVars?: Record<string, string>
  ) {}

  getQueryParams(): KeyValuePair[] {
    const params: KeyValuePair[] = [];

    for (const param of this.specParams) {
      if (param.in !== 'query' || this.disabledParams.includes(param.name)) continue;
      const value = interpolate(this.parameterValues[param.name] ?? '', this.envVars);
      if (value) params.push({ id: param.name, key: param.name, value, enabled: true });
    }

    for (const param of this.customParams) {
      if (param.in !== 'query' || !param.enabled || !param.name) continue;
      params.push({ id: param.id, key: param.name, value: interpolate(param.value, this.envVars), enabled: true });
    }

    return params;
  }

  getHeaderParams(): KeyValuePair[] {
    const params: KeyValuePair[] = [];

    for (const param of this.specParams) {
      if (param.in !== 'header' || this.disabledParams.includes(param.name)) continue;
      const value = interpolate(this.parameterValues[param.name] ?? '', this.envVars);
      if (value) params.push({ id: param.name, key: param.name, value, enabled: true });
    }

    for (const param of this.customParams) {
      if (param.in !== 'header' || !param.enabled || !param.name) continue;
      params.push({ id: param.id, key: param.name, value: interpolate(param.value, this.envVars), enabled: true });
    }

    return params;
  }

  applyPathParams(path: string): string {
    let result = path;

    for (const param of this.specParams) {
      if (param.in !== 'path' || this.disabledParams.includes(param.name)) continue;
      const value = interpolate(this.parameterValues[param.name] ?? '', this.envVars);
      result = result.replace(`{${param.name}}`, encodeURIComponent(value));
    }

    for (const param of this.customParams) {
      if (param.in !== 'path' || !param.enabled || !param.name) continue;
      result = result.replace(`{${param.name}}`, encodeURIComponent(interpolate(param.value, this.envVars)));
    }

    return result;
  }
}

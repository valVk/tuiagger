import { buildManualRequestUrl } from './urlBuilder.js';
import { generateCurl } from './curlGenerator.js';
import type { RequestSpec, KeyValuePair } from '../types/index.js';

export interface BuiltRequest {
  url: string;
  init: RequestInit;
  headers: Record<string, string>;
  curl: string;
}

export class RequestBuilder {
  constructor(private readonly spec: RequestSpec) {}

  build(): BuiltRequest {
    const { method, baseUrl, path, queryParams, headerParams, body, operationSecurity, securitySchemes, authCredentials } = this.spec;

    const headers: Record<string, string> = { Accept: 'application/json' };
    let effectiveQueryParams: KeyValuePair[] = [...queryParams];

    // Inject auth from the first satisfied security requirement
    if (operationSecurity && securitySchemes && authCredentials) {
      outer: for (const requirement of operationSecurity) {
        for (const schemeName of Object.keys(requirement)) {
          const scheme = securitySchemes[schemeName];
          const credential = authCredentials[schemeName];
          if (!scheme || !credential) continue;

          if (scheme.type === 'http') {
            const isBasic = scheme.scheme?.toLowerCase() === 'basic';
            const value = isBasic ? Buffer.from(credential).toString('base64') : credential;
            headers['Authorization'] = `${isBasic ? 'Basic' : 'Bearer'} ${value}`;
          } else if (scheme.type === 'apiKey' && scheme.in === 'header' && scheme.name) {
            headers[scheme.name] = credential;
          } else if (scheme.type === 'apiKey' && scheme.in === 'query' && scheme.name) {
            effectiveQueryParams = [
              ...effectiveQueryParams,
              { id: scheme.name, key: scheme.name, value: credential, enabled: true },
            ];
          }
          break outer;
        }
      }
    }

    // Merge explicit header params
    for (const h of headerParams) {
      if (h.enabled && h.key) headers[h.key] = h.value;
    }

    if (body && !headers['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }

    const url = buildManualRequestUrl(baseUrl, path, effectiveQueryParams);
    const curl = generateCurl(method, url, headers, body);

    const init: RequestInit = { method: method.toUpperCase(), headers };
    if (body && ['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
      init.body = body;
    }

    return { url, init, headers, curl };
  }
}

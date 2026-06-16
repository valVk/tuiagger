import { useState, useCallback } from 'react';
import { buildManualRequestUrl } from '../utils/urlBuilder.js';
import { generateCurl } from '../utils/curlGenerator.js';
import { htmlToPlainText } from '../utils/parser.js';
import type { ResponseState, KeyValuePair, SecurityRequirementObject, SecuritySchemeObject } from '../types/index.js';

interface UseRequestResult {
  response: ResponseState | null;
  loading: boolean;
  error: string | null;
  curl: string | null;
  execute: (
    method: string,
    baseUrl: string,
    path: string,
    queryParams: KeyValuePair[],
    headers: KeyValuePair[],
    body?: string,
    operationSecurity?: SecurityRequirementObject[],
    securitySchemes?: Record<string, SecuritySchemeObject>,
    authCredentials?: Record<string, string>
  ) => Promise<void>;
  clear: () => void;
}

export function useRequest(): UseRequestResult {
  const [response, setResponse] = useState<ResponseState | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [curl, setCurl] = useState<string | null>(null);

  const execute = useCallback(
    async (
      method: string,
      baseUrl: string,
      path: string,
      queryParams: KeyValuePair[],
      headerParams: KeyValuePair[],
      body?: string,
      operationSecurity?: SecurityRequirementObject[],
      securitySchemes?: Record<string, SecuritySchemeObject>,
      authCredentials?: Record<string, string>
    ) => {
      setLoading(true);
      setError(null);
      setResponse(null);

      const startTime = Date.now();
      let url = '';
      let requestBody: string | undefined;
      const headers: Record<string, string> = { Accept: 'application/json' };
      let effectiveQueryParams = queryParams;

      try {
        // Inject auth based on operation security requirements
        if (operationSecurity && securitySchemes && authCredentials) {
          for (const requirement of operationSecurity) {
            for (const schemeName of Object.keys(requirement)) {
              const scheme = securitySchemes[schemeName];
              const credential = authCredentials[schemeName];
              if (!scheme || !credential) continue;

              if (scheme.type === 'http') {
                const isBasic = scheme.scheme?.toLowerCase() === 'basic';
                const prefix = isBasic ? 'Basic' : 'Bearer';
                const value = isBasic
                  ? Buffer.from(credential).toString('base64')
                  : credential;
                headers['Authorization'] = `${prefix} ${value}`;
              } else if (scheme.type === 'apiKey' && scheme.in === 'header' && scheme.name) {
                headers[scheme.name] = credential;
              } else if (scheme.type === 'apiKey' && scheme.in === 'query' && scheme.name) {
                effectiveQueryParams = [
                  ...effectiveQueryParams,
                  { id: scheme.name, key: scheme.name, value: credential, enabled: true },
                ];
              }
              break;
            }
          }
        }

        // Build URL
        url = buildManualRequestUrl(baseUrl, path, effectiveQueryParams);

        // Build headers
        for (const h of headerParams) {
          if (h.enabled && h.key) {
            headers[h.key] = h.value;
          }
        }

        if (body && !headers['Content-Type']) {
          headers['Content-Type'] = 'application/json';
        }

        // Generate curl command
        const curlCmd = generateCurl(method, url, headers, body);
        setCurl(curlCmd);

        // Execute request
        const fetchOptions: RequestInit = {
          method: method.toUpperCase(),
          headers,
        };

        if (body && ['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
          fetchOptions.body = body;
          requestBody = body;
        }

        const res = await fetch(url, fetchOptions);
        const endTime = Date.now();

        // Parse response
        const responseHeaders: Record<string, string> = {};
        res.headers.forEach((value, key) => {
          responseHeaders[key] = value;
        });

        let responseBody: string;
        const contentType = res.headers.get('content-type') || '';

        if (contentType.includes('application/json')) {
          const json = await res.json();
          responseBody = JSON.stringify(json, null, 2);
        } else {
          const htmlText = await res.text();
          responseBody = htmlToPlainText(htmlText);
        }

        setResponse({
          status: res.status,
          statusText: res.statusText,
          headers: responseHeaders,
          body: responseBody,
          time: endTime - startTime,
          requestMethod: method.toUpperCase(),
          requestUrl: url,
          requestHeaders: headers,
          requestBody,
        });
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Request failed';
        setError(message);
        setResponse({
          status: 0,
          statusText: 'Error',
          headers: {},
          body: '',
          time: Date.now() - startTime,
          error: message,
          requestMethod: method.toUpperCase(),
          requestUrl: url,
          requestHeaders: headers,
          requestBody,
        });
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const clear = useCallback(() => {
    setResponse(null);
    setError(null);
    setCurl(null);
  }, []);

  return {
    response,
    loading,
    error,
    curl,
    execute,
    clear,
  };
}

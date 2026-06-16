import { useState, useCallback } from 'react';
import { RequestBuilder } from '../utils/requestBuilder.js';
import { htmlToPlainText } from '../utils/parser.js';
import { useServices } from '../contexts/ServicesContext.js';
import type { ResponseState, RequestSpec } from '../types/index.js';

interface UseRequestResult {
  response: ResponseState | null;
  loading: boolean;
  error: string | null;
  curl: string | null;
  execute: (spec: RequestSpec) => Promise<void>;
  clear: () => void;
}

export function useRequest(): UseRequestResult {
  const { httpClient } = useServices();
  const [response, setResponse] = useState<ResponseState | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [curl, setCurl] = useState<string | null>(null);

  const execute = useCallback(async (spec: RequestSpec) => {
    setLoading(true);
    setError(null);
    setResponse(null);

    const startTime = Date.now();

    try {
      const { url, init, headers, curl: curlCmd } = new RequestBuilder(spec).build();
      setCurl(curlCmd);

      const res = await httpClient.fetch(url, init);
      const endTime = Date.now();

      const responseHeaders: Record<string, string> = {};
      res.headers.forEach((value, key) => { responseHeaders[key] = value; });

      let responseBody: string;
      const contentType = res.headers.get('content-type') ?? '';

      if (contentType.includes('application/json')) {
        const json = await res.json();
        responseBody = JSON.stringify(json, null, 2);
      } else {
        responseBody = htmlToPlainText(await res.text());
      }

      setResponse({
        status: res.status,
        statusText: res.statusText,
        headers: responseHeaders,
        body: responseBody,
        time: endTime - startTime,
        requestMethod: spec.method.toUpperCase(),
        requestUrl: url,
        requestHeaders: headers,
        requestBody: init.body as string | undefined,
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
        requestMethod: spec.method.toUpperCase(),
        requestUrl: '',
        requestHeaders: {},
      });
    } finally {
      setLoading(false);
    }
  }, [httpClient]);

  const clear = useCallback(() => {
    setResponse(null);
    setError(null);
    setCurl(null);
  }, []);

  return { response, loading, error, curl, execute, clear };
}

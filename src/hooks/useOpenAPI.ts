import { useState, useEffect } from 'react';
import { parseOpenAPISpec, type ParsedSpec } from '../utils/parser.js';

interface UseOpenAPIResult {
  spec: ParsedSpec | null;
  loading: boolean;
  error: string | null;
  reload: () => void;
}

export function useOpenAPI(source: string): UseOpenAPIResult {
  const [spec, setSpec] = useState<ParsedSpec | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    setError(null);

    try {
      const parsed = await parseOpenAPISpec(source);
      setSpec(parsed);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to parse OpenAPI spec';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [source]);

  return {
    spec,
    loading,
    error,
    reload: load,
  };
}

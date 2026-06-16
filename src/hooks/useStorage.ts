import { useState, useEffect, useCallback } from 'react';

export function useStorage<T>(
  load: () => Promise<T>,
  save: (value: T) => Promise<void>,
  initial: T
): [T, (value: T) => void, boolean] {
  const [value, setValue] = useState<T>(initial);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    load().then(v => {
      setValue(v);
      setLoading(false);
    });
  }, []);

  const set = useCallback((next: T) => {
    setValue(next);
    void save(next);
  }, []);

  return [value, set, loading];
}

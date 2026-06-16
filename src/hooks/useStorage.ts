import { useState, useEffect, useCallback } from 'react';

type Updater<T> = T | ((prev: T) => T);

export function useStorage<T>(
  load: () => Promise<T>,
  save: (value: T) => Promise<void>,
  initial: T
): [T, (updater: Updater<T>) => void, boolean] {
  const [value, setValue] = useState<T>(initial);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    load().then(v => {
      setValue(v);
      setLoading(false);
    });
  }, []);

  const set = useCallback((updater: Updater<T>) => {
    setValue(prev => {
      const next = typeof updater === 'function' ? (updater as (prev: T) => T)(prev) : updater;
      void save(next);
      return next;
    });
  }, []);

  return [value, set, loading];
}

import { useState, useCallback, useEffect } from 'react';
import { loadEnvironments, saveEnvironments } from '../utils/storage.js';
import type { EnvironmentsStore } from '../types/index.js';

export function useEnvironments() {
  const [store, setStore] = useState<EnvironmentsStore>({ version: '1.0', environments: [], activeIndex: -1 });

  useEffect(() => {
    loadEnvironments().then(setStore);
  }, []);

  const save = useCallback((next: EnvironmentsStore) => {
    setStore(next);
    saveEnvironments(next);
  }, []);

  const activeEnv = store.environments[store.activeIndex] ?? null;

  const setActive = useCallback((index: number) => {
    save({ ...store, activeIndex: index });
  }, [store, save]);

  const addEnvironment = useCallback((name: string) => {
    save({ ...store, environments: [...store.environments, { name, variables: {} }] });
  }, [store, save]);

  const deleteEnvironment = useCallback((index: number) => {
    const envs = store.environments.filter((_, i) => i !== index);
    const newActive = store.activeIndex >= envs.length ? envs.length - 1 : store.activeIndex;
    save({ ...store, environments: envs, activeIndex: newActive });
  }, [store, save]);

  const setVariable = useCallback((envIndex: number, key: string, value: string) => {
    const envs = store.environments.map((e, i) =>
      i === envIndex ? { ...e, variables: { ...e.variables, [key]: value } } : e
    );
    save({ ...store, environments: envs });
  }, [store, save]);

  const deleteVariable = useCallback((envIndex: number, key: string) => {
    const envs = store.environments.map((e, i) => {
      if (i !== envIndex) return e;
      const vars = { ...e.variables };
      delete vars[key];
      return { ...e, variables: vars };
    });
    save({ ...store, environments: envs });
  }, [store, save]);

  return { store, activeEnv, setActive, addEnvironment, deleteEnvironment, setVariable, deleteVariable };
}

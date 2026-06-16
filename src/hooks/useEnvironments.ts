import { useCallback } from 'react';
import { loadEnvironments, saveEnvironments } from '../utils/persistence.js';
import { useStorage } from './useStorage.js';
import type { EnvironmentsStore } from '../types/index.js';

const INITIAL: EnvironmentsStore = { version: '1.0', environments: [], activeIndex: -1 };

export function useEnvironments() {
  const [store, setStore] = useStorage(loadEnvironments, saveEnvironments, INITIAL);

  const activeEnv = store.environments[store.activeIndex] ?? null;

  const setActive = useCallback((index: number) => {
    setStore({ ...store, activeIndex: index });
  }, [store, setStore]);

  const addEnvironment = useCallback((name: string) => {
    setStore({ ...store, environments: [...store.environments, { name, variables: {} }] });
  }, [store, setStore]);

  const deleteEnvironment = useCallback((index: number) => {
    const envs = store.environments.filter((_, i) => i !== index);
    const newActive = store.activeIndex >= envs.length ? envs.length - 1 : store.activeIndex;
    setStore({ ...store, environments: envs, activeIndex: newActive });
  }, [store, setStore]);

  const setVariable = useCallback((envIndex: number, key: string, value: string) => {
    const envs = store.environments.map((e, i) =>
      i === envIndex ? { ...e, variables: { ...e.variables, [key]: value } } : e
    );
    setStore({ ...store, environments: envs });
  }, [store, setStore]);

  const deleteVariable = useCallback((envIndex: number, key: string) => {
    const envs = store.environments.map((e, i) => {
      if (i !== envIndex) return e;
      const vars = { ...e.variables };
      delete vars[key];
      return { ...e, variables: vars };
    });
    setStore({ ...store, environments: envs });
  }, [store, setStore]);

  return { store, activeEnv, setActive, addEnvironment, deleteEnvironment, setVariable, deleteVariable };
}

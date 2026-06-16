import { useState } from 'react';
import { useInput } from 'ink';
import type { EnvironmentsStore } from '../types/index.js';

type EnvView = 'list' | 'edit';
type VarField = 'key' | 'value';

interface UseEnvironmentsKeyboardOptions {
  environments: EnvironmentsStore;
  isActive: boolean;
  onSetActive: (index: number) => void;
  onAddEnvironment: (name: string) => void;
  onDeleteEnvironment: (index: number) => void;
  onSetVariable: (envIndex: number, key: string, value: string) => void;
  onDeleteVariable: (envIndex: number, key: string) => void;
}

export function useEnvironmentsKeyboard({
  environments,
  isActive,
  onSetActive,
  onAddEnvironment,
  onDeleteEnvironment,
  onSetVariable,
  onDeleteVariable,
}: UseEnvironmentsKeyboardOptions) {
  const [envView, setEnvView] = useState<EnvView>('list');
  const [envCursor, setEnvCursor] = useState(0);
  const [addingEnv, setAddingEnv] = useState(false);
  const [newEnvName, setNewEnvName] = useState('');
  const [varCursor, setVarCursor] = useState(0);
  const [insertingVar, setInsertingVar] = useState(false);
  const [varField, setVarField] = useState<VarField>('key');
  const [editingVarKey, setEditingVarKey] = useState<string | null>(null);
  const [varKeyInput, setVarKeyInput] = useState('');
  const [varValueInput, setVarValueInput] = useState('');

  const isInsertMode = addingEnv || insertingVar;
  const envList = environments.environments;

  useInput((input, key) => {
    if (!isActive) return;

    if (addingEnv) {
      if (key.return && newEnvName.trim()) {
        onAddEnvironment(newEnvName.trim());
        setEnvCursor(envList.length);
        setAddingEnv(false);
        setNewEnvName('');
      } else if (key.escape) {
        setAddingEnv(false);
        setNewEnvName('');
      }
      return;
    }

    if (insertingVar) {
      if (key.tab) { setVarField(f => f === 'key' ? 'value' : 'key'); return; }
      if (key.escape) {
        setInsertingVar(false);
        setEditingVarKey(null);
        setVarKeyInput('');
        setVarValueInput('');
        return;
      }
      if (key.return) {
        if (varKeyInput.trim()) onSetVariable(envCursor, varKeyInput.trim(), varValueInput);
        setInsertingVar(false);
        setEditingVarKey(null);
        setVarKeyInput('');
        setVarValueInput('');
      }
      return;
    }

    if (envView === 'list') {
      if (input === 'j' || key.downArrow) { setEnvCursor(p => Math.min(p + 1, envList.length - 1)); return; }
      if (input === 'k' || key.upArrow)   { setEnvCursor(p => Math.max(p - 1, 0)); return; }
      if (key.return && envList.length > 0) { onSetActive(envCursor); return; }
      if (input === 'e' && envList.length > 0) { setEnvView('edit'); setVarCursor(0); return; }
      if (input === 'n') { setAddingEnv(true); setNewEnvName(''); return; }
      if (input === 'x' && envList.length > 0) {
        onDeleteEnvironment(envCursor);
        setEnvCursor(c => Math.max(0, c - 1));
        return;
      }
    }

    if (envView === 'edit') {
      const env = envList[envCursor];
      if (!env) return;
      const varEntries = Object.entries(env.variables);
      const totalVarRows = varEntries.length + 1;

      if (input === 'j' || key.downArrow) { setVarCursor(p => Math.min(p + 1, totalVarRows - 1)); return; }
      if (input === 'k' || key.upArrow)   { setVarCursor(p => Math.max(p - 1, 0)); return; }

      if (input === 'i') {
        if (varCursor < varEntries.length) {
          const [k, v] = varEntries[varCursor];
          setEditingVarKey(k);
          setVarKeyInput(k);
          setVarValueInput(v);
        } else {
          setEditingVarKey(null);
          setVarKeyInput('');
          setVarValueInput('');
        }
        setVarField('key');
        setInsertingVar(true);
        return;
      }

      if (input === 'x' && varCursor < varEntries.length) {
        onDeleteVariable(envCursor, varEntries[varCursor][0]);
        setVarCursor(c => Math.max(0, c - 1));
        return;
      }
    }
  }, { isActive });

  return {
    envView, setEnvView,
    envCursor, varCursor,
    addingEnv, newEnvName, setNewEnvName,
    insertingVar, varField, editingVarKey,
    varKeyInput, setVarKeyInput,
    varValueInput, setVarValueInput,
    isInsertMode,
  };
}

import { useState, useEffect } from 'react';
import { useInput } from 'ink';
import { randomUUID } from 'crypto';
import type { ParameterObject, CustomParameter } from '../types/index.js';

export interface RowItem {
  type: 'spec' | 'custom' | 'addNew';
  specParam?: ParameterObject;
  customParam?: CustomParameter;
}

interface UseParamNavigationOptions {
  rows: RowItem[];
  values: Record<string, string>;
  isTryItMode: boolean;
  isActive: boolean;
  customParams: CustomParameter[];
  disabledParams: string[];
  resetKey?: number;
  onChange?: (name: string, value: string) => void;
  onCustomParamsChange?: (params: CustomParameter[]) => void;
  onDisabledParamsChange?: (disabled: string[]) => void;
  onInsertModeChange?: (v: boolean) => void;
  onFocusChange?: (index: number) => void;
  onTabOut?: (totalRows: number) => void;
  onTabBack?: () => void;
}

interface UseParamNavigationResult {
  selectedRow: number;
  safeSelectedRow: number;
  insertMode: boolean;
  editingField: 'name' | 'value' | null;
  addingParam: boolean;
  newParamName: string;
  newParamValue: string;
  newParamType: 'query' | 'path';
  editingNewField: 'name' | 'value';
  setNewParamName: (v: string) => void;
  setNewParamValue: (v: string) => void;
}

export function useParamNavigation({
  rows,
  values,
  isTryItMode,
  isActive,
  customParams,
  disabledParams,
  resetKey,
  onChange,
  onCustomParamsChange,
  onDisabledParamsChange,
  onInsertModeChange,
  onFocusChange,
  onTabOut,
  onTabBack,
}: UseParamNavigationOptions): UseParamNavigationResult {
  const [selectedRow, setSelectedRow] = useState(0);
  const [insertMode, setInsertMode] = useState(false);
  const [editingField, setEditingField] = useState<'name' | 'value' | null>(null);
  const [addingParam, setAddingParam] = useState(false);
  const [newParamName, setNewParamName] = useState('');
  const [newParamValue, setNewParamValue] = useState('');
  const [newParamType, setNewParamType] = useState<'query' | 'path'>('query');
  const [editingNewField, setEditingNewField] = useState<'name' | 'value'>('name');

  useEffect(() => {
    if (resetKey !== undefined && resetKey > 0) {
      setSelectedRow(0);
    }
  }, [resetKey]);

  const safeSelectedRow = Math.min(selectedRow, Math.max(0, rows.length - 1));
  const currentRow = rows[safeSelectedRow] || null;

  const enterInsertMode = (v: boolean) => {
    setInsertMode(v);
    onInsertModeChange?.(v);
  };

  useInput(
    (input, key) => {
      if (!isTryItMode || !isActive) return;

      if (insertMode) {
        if (key.escape) {
          enterInsertMode(false);
          setEditingField(null);
          setAddingParam(false);
          setNewParamName('');
          setNewParamValue('');
          setEditingNewField('name');
          return;
        }

        if (key.tab && currentRow?.type === 'custom') {
          setEditingField(prev => prev === 'name' ? 'value' : 'name');
          return;
        }

        if (key.tab && addingParam) {
          setEditingNewField(prev => prev === 'name' ? 'value' : 'name');
          return;
        }

        if (key.return && addingParam && newParamName.trim()) {
          const newParam: CustomParameter = {
            id: randomUUID(),
            name: newParamName.trim(),
            value: newParamValue,
            in: newParamType,
            enabled: true,
          };
          onCustomParamsChange?.([...customParams, newParam]);
          setAddingParam(false);
          setNewParamName('');
          setNewParamValue('');
          setEditingNewField('name');
          enterInsertMode(false);
          setSelectedRow(rows.length - 1);
          return;
        }

        if (currentRow?.type === 'spec') {
          const enumValues = currentRow.specParam!.schema?.enum as string[] | undefined;
          if (enumValues?.length) {
            const currentValue = values[currentRow.specParam!.name] || '';
            const idx = enumValues.indexOf(currentValue);
            if (key.leftArrow) {
              const next = idx <= 0 ? enumValues.length - 1 : idx - 1;
              onChange?.(currentRow.specParam!.name, enumValues[next]);
              return;
            }
            if (key.rightArrow) {
              const next = (idx + 1) % enumValues.length;
              onChange?.(currentRow.specParam!.name, enumValues[next]);
              return;
            }
            if (key.return) {
              enterInsertMode(false);
              setEditingField(null);
              return;
            }
          }
        }

        return;
      }

      // Normal mode
      if (input === 'j' || key.downArrow) {
        if (safeSelectedRow < rows.length - 1) {
          const newRow = safeSelectedRow + 1;
          setSelectedRow(newRow);
          onFocusChange?.(newRow);
        } else {
          onTabOut?.(rows.length);
        }
        return;
      }

      if (input === 'k' || key.upArrow) {
        if (safeSelectedRow > 0) {
          const newRow = safeSelectedRow - 1;
          setSelectedRow(newRow);
          onFocusChange?.(newRow);
        } else {
          onTabBack?.();
        }
        return;
      }

      if (input === 'i') {
        if (currentRow?.type === 'spec') {
          if (!disabledParams.includes(currentRow.specParam!.name)) {
            const enumVals = currentRow.specParam!.schema?.enum as string[] | undefined;
            if (enumVals?.length && !values[currentRow.specParam!.name]) {
              onChange?.(currentRow.specParam!.name, enumVals[0]);
            }
            enterInsertMode(true);
            setEditingField('value');
          }
          return;
        }
        if (currentRow?.type === 'custom') {
          enterInsertMode(true);
          setEditingField('value');
          return;
        }
        if (currentRow?.type === 'addNew') {
          enterInsertMode(true);
          setAddingParam(true);
          setNewParamName('');
          setNewParamValue('');
          setEditingNewField('name');
          return;
        }
      }

      if (input === 'd' && currentRow?.type === 'spec') {
        const paramName = currentRow.specParam!.name;
        if (disabledParams.includes(paramName)) {
          onDisabledParamsChange?.(disabledParams.filter(n => n !== paramName));
        } else {
          onDisabledParamsChange?.([...disabledParams, paramName]);
        }
        return;
      }

      if (input === 'x' && currentRow?.type === 'custom') {
        const paramId = currentRow.customParam!.id;
        onCustomParamsChange?.(customParams.filter(p => p.id !== paramId));
        if (safeSelectedRow >= rows.length - 1) {
          setSelectedRow(Math.max(0, rows.length - 2));
        }
        return;
      }

      if (input === 'c') {
        const types: Array<'query' | 'path'> = ['query', 'path'];
        if (currentRow?.type === 'custom') {
          const paramId = currentRow.customParam!.id;
          onCustomParamsChange?.(
            customParams.map(p => {
              if (p.id === paramId) {
                const cur = types.includes(p.in as 'query' | 'path') ? p.in as 'query' | 'path' : 'query';
                const nextIndex = (types.indexOf(cur) + 1) % types.length;
                return { ...p, in: types[nextIndex] };
              }
              return p;
            })
          );
          return;
        }
        if (currentRow?.type === 'addNew') {
          setNewParamType(prev => types[(types.indexOf(prev) + 1) % types.length]);
          return;
        }
      }
    },
    { isActive: isTryItMode && isActive }
  );

  return {
    selectedRow,
    safeSelectedRow,
    insertMode,
    editingField,
    addingParam,
    newParamName,
    newParamValue,
    newParamType,
    editingNewField,
    setNewParamName,
    setNewParamValue,
  };
}

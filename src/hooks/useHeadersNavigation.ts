import { useState } from 'react';
import { useInput } from 'ink';
import { randomUUID } from 'crypto';
import type { CustomParameter } from '../types/index.js';

interface UseHeadersNavigationOptions {
  headers: CustomParameter[];
  isActive: boolean;
  onChange: (headers: CustomParameter[]) => void;
  onTabOut?: () => void;
  onTabBack?: () => void;
  onInsertModeChange?: (v: boolean) => void;
}

interface UseHeadersNavigationResult {
  cursor: number;
  insertMode: boolean;
  editingField: 'name' | 'value';
  newName: string;
  newValue: string;
  editName: string;
  editValue: string;
  editingExisting: boolean;
  setNewName: (v: string) => void;
  setNewValue: (v: string) => void;
  setEditName: (v: string) => void;
  setEditValue: (v: string) => void;
}

export function useHeadersNavigation({
  headers,
  isActive,
  onChange,
  onTabOut,
  onTabBack,
  onInsertModeChange,
}: UseHeadersNavigationOptions): UseHeadersNavigationResult {
  const [cursor, setCursor] = useState(0);
  const [insertMode, setInsertMode] = useState(false);
  const [editingField, setEditingField] = useState<'name' | 'value'>('name');
  const [newName, setNewName] = useState('');
  const [newValue, setNewValue] = useState('');
  const [editName, setEditName] = useState('');
  const [editValue, setEditValue] = useState('');
  const [editingExisting, setEditingExisting] = useState(false);

  const rows = headers.length;
  const totalRows = rows + 1;

  const enterInsert = (val: boolean) => {
    setInsertMode(val);
    onInsertModeChange?.(val);
  };

  useInput((input, key) => {
    if (!isActive) return;

    if (insertMode) {
      if (key.tab) {
        setEditingField(f => f === 'name' ? 'value' : 'name');
        return;
      }
      if (key.escape) {
        enterInsert(false);
        setEditingExisting(false);
        setNewName('');
        setNewValue('');
        return;
      }
      if (key.return) {
        if (editingExisting) {
          onChange(headers.map((h, i) =>
            i === cursor ? { ...h, name: editName, value: editValue } : h
          ));
          enterInsert(false);
          setEditingExisting(false);
        } else {
          if (newName.trim()) {
            onChange([...headers, { id: randomUUID(), name: newName.trim(), value: newValue, in: 'header', enabled: true }]);
            setCursor(rows);
          }
          enterInsert(false);
          setNewName('');
          setNewValue('');
        }
        return;
      }
      return;
    }

    if (input === 'j' || key.downArrow) {
      if (cursor < totalRows - 1) setCursor(c => c + 1);
      else onTabOut?.();
      return;
    }
    if (input === 'k' || key.upArrow) {
      if (cursor > 0) setCursor(c => c - 1);
      else onTabBack?.();
      return;
    }
    if (key.tab) { onTabOut?.(); return; }
    if (key.escape) { onTabBack?.(); return; }

    if (input === 'i') {
      if (cursor < rows) {
        setEditName(headers[cursor].name);
        setEditValue(headers[cursor].value);
        setEditingField('name');
        setEditingExisting(true);
        enterInsert(true);
      } else {
        setNewName('');
        setNewValue('');
        setEditingField('name');
        setEditingExisting(false);
        enterInsert(true);
      }
      return;
    }

    if (input === 'x' && cursor < rows) {
      const updated = headers.filter((_, i) => i !== cursor);
      onChange(updated);
      setCursor(c => Math.min(c, updated.length));
      return;
    }

    if (input === 'd' && cursor < rows) {
      onChange(headers.map((h, i) => i === cursor ? { ...h, enabled: !h.enabled } : h));
      return;
    }
  }, { isActive });

  return { cursor, insertMode, editingField, newName, newValue, editName, editValue, editingExisting, setNewName, setNewValue, setEditName, setEditValue };
}

import { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { randomUUID } from 'crypto';
import type { CustomParameter } from '../types/index.js';

interface HeadersSectionProps {
  headers: CustomParameter[];
  onChange: (headers: CustomParameter[]) => void;
  isActive?: boolean;
  onTabOut?: () => void;
  onTabBack?: () => void;
  onInsertModeChange?: (inserting: boolean) => void;
}

const CURSOR_WIDTH = 3;
const NAME_WIDTH = 25;
const VALUE_WIDTH = 28;

export function HeadersSection({
  headers,
  onChange,
  isActive = false,
  onTabOut,
  onTabBack,
  onInsertModeChange,
}: HeadersSectionProps) {
  const [cursor, setCursor] = useState(0);
  const [insertMode, setInsertMode] = useState(false);
  const [editingField, setEditingField] = useState<'name' | 'value'>('name');
  const [newName, setNewName] = useState('');
  const [newValue, setNewValue] = useState('');
  const [editName, setEditName] = useState('');
  const [editValue, setEditValue] = useState('');
  const [editingExisting, setEditingExisting] = useState(false);

  const rows = headers.length;
  const totalRows = rows + 1; // +1 for addNew

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

  const isCursorOnAdd = isActive && cursor === rows;

  return (
    <Box flexDirection="column" width="100%">
      {/* Section header */}
      <Box>
        <Text bold>HEADERS</Text>
        {isActive && !insertMode && (
          <Text dimColor> j/k: move | i: edit | d: toggle | x: del</Text>
        )}
        {isActive && insertMode && (
          <Text dimColor> Tab: switch field | Enter: confirm | Esc: cancel</Text>
        )}
      </Box>

      {/* Table header */}
      <Box>
        <Box width={CURSOR_WIDTH} flexShrink={0} />
        <Box width={NAME_WIDTH} flexShrink={0}><Text bold dimColor>NAME</Text></Box>
        <Box width={VALUE_WIDTH} flexShrink={0}><Text bold dimColor>VALUE</Text></Box>
      </Box>

      {/* Header rows */}
      {headers.map((h, i) => {
        const isCur = isActive && i === cursor;
        const isEditingThis = insertMode && editingExisting && i === cursor;

        return (
          <Box key={h.id}>
            <Box width={CURSOR_WIDTH} flexShrink={0}>
              <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
            </Box>
            <Box width={NAME_WIDTH} flexShrink={0}>
              {isEditingThis && editingField === 'name'
                ? <TextInput value={editName} onChange={setEditName} focus={true} />
                : <Text color={isCur ? 'cyan' : undefined} dimColor={!h.enabled}>{h.name || '-'}</Text>
              }
            </Box>
            <Box width={VALUE_WIDTH} flexShrink={0}>
              {isEditingThis && editingField === 'value'
                ? <TextInput value={editValue} onChange={setEditValue} focus={true} />
                : <Text color="green" dimColor={!h.enabled}>{h.value || '-'}</Text>
              }
            </Box>
          </Box>
        );
      })}

      {/* Add new row */}
      {(() => {
        const isAddingNew = insertMode && !editingExisting;
        return (
          <Box>
            <Box width={CURSOR_WIDTH} flexShrink={0}>
              <Text color={isCursorOnAdd ? 'cyan' : 'gray'}>{isCursorOnAdd ? '> ' : '  '}</Text>
            </Box>
            {isAddingNew ? (
              <>
                <Box width={NAME_WIDTH} flexShrink={0}>
                  {editingField === 'name'
                    ? <TextInput value={newName} onChange={setNewName} focus={true} placeholder="header-name" />
                    : <Text color="cyan">{newName || '-'}</Text>
                  }
                </Box>
                <Box width={VALUE_WIDTH} flexShrink={0}>
                  {editingField === 'value'
                    ? <TextInput value={newValue} onChange={setNewValue} focus={true} placeholder="value" />
                    : <Text dimColor>-</Text>
                  }
                </Box>
              </>
            ) : (
              <Text color={isCursorOnAdd ? 'cyan' : undefined} dimColor={!isCursorOnAdd}>
                {isCursorOnAdd ? '[ i: add header ]' : '[ + ]'}
              </Text>
            )}
          </Box>
        );
      })()}
    </Box>
  );
}

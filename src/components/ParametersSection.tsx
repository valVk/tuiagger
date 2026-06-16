import { useState, useMemo, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { randomUUID } from 'crypto';
import type { ParameterObject, CustomParameter } from '../types/index.js';
import { SpecParamRow } from './SpecParamRow.js';

interface ParametersSectionProps {
  parameters: ParameterObject[];
  isTryItMode?: boolean;
  values?: Record<string, string>;
  onChange?: (name: string, value: string) => void;
  isActive?: boolean;
  onFocusChange?: (index: number) => void;
  onTabOut?: (totalRows: number) => void;
  onTabBack?: () => void;
  resetKey?: number;
  customParams?: CustomParameter[];
  onCustomParamsChange?: (params: CustomParameter[]) => void;
  disabledParams?: string[];
  onDisabledParamsChange?: (disabled: string[]) => void;
  onInsertModeChange?: (inserting: boolean) => void;
}

// Row-based navigation - each param is one row
interface RowItem {
  type: 'spec' | 'custom' | 'addNew';
  specParam?: ParameterObject;
  customParam?: CustomParameter;
}

export function ParametersSection({
  parameters,
  isTryItMode = false,
  values = {},
  onChange,
  isActive = false,
  onFocusChange,
  onTabOut,
  onTabBack,
  resetKey,
  customParams = [],
  onCustomParamsChange,
  disabledParams = [],
  onDisabledParamsChange,
  onInsertModeChange,
}: ParametersSectionProps) {
  const [selectedRow, setSelectedRow] = useState(0);
  const [insertMode, setInsertMode] = useState(false);
  const [editingField, setEditingField] = useState<'name' | 'value' | null>(null);

  useEffect(() => {
    if (resetKey !== undefined && resetKey > 0) {
      setSelectedRow(0);
    }
  }, [resetKey]);

  // For adding new params
  const [addingParam, setAddingParam] = useState(false);
  const [newParamName, setNewParamName] = useState('');
  const [newParamValue, setNewParamValue] = useState('');
  const [newParamType, setNewParamType] = useState<'query' | 'path'>('query');
  const [editingNewField, setEditingNewField] = useState<'name' | 'value'>('name');

  const enterInsertMode = (v: boolean) => {
    setInsertMode(v);
    onInsertModeChange?.(v);
  };

  // Build row list
  const rows = useMemo((): RowItem[] => {
    const items: RowItem[] = [];

    // Spec params (required first, then optional)
    const requiredParams = parameters.filter(p => p.required);
    const optionalParams = parameters.filter(p => !p.required);

    for (const param of [...requiredParams, ...optionalParams]) {
      items.push({ type: 'spec', specParam: param });
    }

    // Custom params
    for (const param of customParams) {
      items.push({ type: 'custom', customParam: param });
    }

    // Add new row (only in try-it mode)
    if (isTryItMode) {
      items.push({ type: 'addNew' });
    }

    return items;
  }, [parameters, customParams, isTryItMode]);

  const currentRow = rows[selectedRow] || null;
  const safeSelectedRow = Math.min(selectedRow, Math.max(0, rows.length - 1));

  useInput(
    (input, key) => {
      if (!isTryItMode || !isActive) return;

      // === INSERT MODE ===
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

        // Tab to switch between fields in custom params
        if (key.tab && currentRow?.type === 'custom') {
          setEditingField(prev => prev === 'name' ? 'value' : 'name');
          return;
        }

        // Tab to switch between name/value when adding new param
        if (key.tab && addingParam) {
          setEditingNewField(prev => prev === 'name' ? 'value' : 'name');
          return;
        }

        // Enter confirms new param
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

        // Enum cycling — left/right when current spec param has enum values
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

        // Let TextInput handle other input — no 'c' binding in insert mode
        return;
      }

      // === NORMAL MODE ===

      // j / Down — next row
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

      // k / Up — previous row, or tab back to section above
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

      // 'i' to enter edit mode
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

      // 'd' to toggle disable (spec params only)
      if (input === 'd' && currentRow?.type === 'spec') {
        const paramName = currentRow.specParam!.name;
        if (disabledParams.includes(paramName)) {
          onDisabledParamsChange?.(disabledParams.filter(n => n !== paramName));
        } else {
          onDisabledParamsChange?.([...disabledParams, paramName]);
        }
        return;
      }

      // 'x' to delete (custom params only)
      if (input === 'x' && currentRow?.type === 'custom') {
        const paramId = currentRow.customParam!.id;
        onCustomParamsChange?.(customParams.filter(p => p.id !== paramId));
        if (safeSelectedRow >= rows.length - 1) {
          setSelectedRow(Math.max(0, rows.length - 2));
        }
        return;
      }

      // 'c' to cycle type (custom params and addNew row, normal mode only)
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

  // Column widths
  const nameWidth = 25;
  const valueWidth = 25;
  const typeWidth = 12;

  const isAddNewSelected = isActive && currentRow?.type === 'addNew';
  const currentIsEnum = insertMode
    && currentRow?.type === 'spec'
    && !!(currentRow.specParam?.schema?.enum as string[] | undefined)?.length;

  return (
    <Box flexDirection="column" width="100%">
      <Box>
        <Text bold>PARAMETERS</Text>
        {isTryItMode && isActive && (
          <Text dimColor>
            {insertMode
              ? currentIsEnum
                ? ' [EDIT] ←/→: cycle | Enter/Esc: done'
                : ' [EDIT] Tab: next field | Esc: done'
              : ' j/k: move | i: edit | d: toggle | x: del | c: type'}
          </Text>
        )}
      </Box>

      {/* Table header */}
      <Box width="100%">
        <Box width={3} flexShrink={0} />
        <Box width={nameWidth} flexShrink={0}><Text dimColor bold>NAME</Text></Box>
        <Box width={valueWidth} flexShrink={0}><Text dimColor bold>VALUE</Text></Box>
        <Box width={typeWidth} flexShrink={0}><Text dimColor bold>TYPE</Text></Box>
        <Box flexGrow={1}><Text dimColor bold>DESCRIPTION</Text></Box>
      </Box>

      <Box borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" />

      {/* Flat unified param list */}
      {rows.map((row, rowIndex) => {
        const isRowSelected = isTryItMode && isActive && safeSelectedRow === rowIndex;
        const isRowEditing = isActive && insertMode && safeSelectedRow === rowIndex;

        if (row.type === 'spec') {
          return (
            <SpecParamRow
              key={`${row.specParam!.in}-${row.specParam!.name}`}
              param={row.specParam!}
              value={values[row.specParam!.name] || ''}
              isSelected={isRowSelected}
              isEditing={isRowEditing}
              isDisabled={disabledParams.includes(row.specParam!.name)}
              onChange={onChange}
              nameWidth={nameWidth}
              valueWidth={valueWidth}
              typeWidth={typeWidth}
            />
          );
        }

        if (row.type === 'custom') {
          return (
            <CustomParamRow
              key={row.customParam!.id}
              param={row.customParam!}
              isSelected={isRowSelected}
              isEditing={isRowEditing}
              editingField={editingField}
              onNameChange={(name) => onCustomParamsChange?.(customParams.map(p => p.id === row.customParam!.id ? { ...p, name } : p))}
              onValueChange={(value) => onCustomParamsChange?.(customParams.map(p => p.id === row.customParam!.id ? { ...p, value } : p))}
              nameWidth={nameWidth}
              valueWidth={valueWidth}
              typeWidth={typeWidth}
            />
          );
        }

        if (row.type === 'addNew') {
          return (
            <Box key="addNew" width="100%">
              <Box width={3} flexShrink={0}>
                <Text color={isAddNewSelected ? 'cyan' : undefined} dimColor={!isAddNewSelected}>{isAddNewSelected ? '>' : ' '}</Text>
              </Box>
              <Box width={nameWidth} flexShrink={0}>
                {addingParam && insertMode && editingNewField === 'name' ? (
                  <TextInput value={newParamName} onChange={setNewParamName} placeholder="param name" focus={true} />
                ) : addingParam && insertMode ? (
                  <Text color="cyan">{newParamName || '-'}</Text>
                ) : (
                  <Text color={isAddNewSelected ? 'cyan' : undefined} dimColor={!isAddNewSelected}>
                    {isAddNewSelected ? '[ i: add parameter ]' : '[ + ]'}
                  </Text>
                )}
              </Box>
              <Box width={valueWidth} flexShrink={0}>
                {addingParam && insertMode && editingNewField === 'value' ? (
                  <TextInput value={newParamValue} onChange={setNewParamValue} placeholder="value" focus={true} />
                ) : addingParam && insertMode ? (
                  <Text dimColor>{newParamValue || '-'}</Text>
                ) : isAddNewSelected ? (
                  <Text dimColor>c: type</Text>
                ) : null}
              </Box>
              {((addingParam && insertMode) || isAddNewSelected) && (
                <Box width={typeWidth} flexShrink={0}>
                  <Text color="yellow">{newParamType}</Text>
                </Box>
              )}
            </Box>
          );
        }

        return null;
      })}
    </Box>
  );
}

interface CustomParamRowProps {
  param: CustomParameter;
  isSelected: boolean;
  isEditing: boolean;
  editingField: 'name' | 'value' | null;
  onNameChange: (name: string) => void;
  onValueChange: (value: string) => void;
  nameWidth: number;
  valueWidth: number;
  typeWidth: number;
}

function CustomParamRow({
  param,
  isSelected,
  isEditing,
  editingField,
  onNameChange,
  onValueChange,
  nameWidth,
  valueWidth,
  typeWidth,
}: CustomParamRowProps) {
  return (
    <Box width="100%">
      {/* Selection cursor — space reserved always */}
      <Box width={3} flexShrink={0}>
        <Text color="cyan">{isSelected ? '>' : ' '}</Text>
      </Box>

      {/* Name - TextInput ONLY when editing name */}
      <Box width={nameWidth} flexShrink={0}>
        {isEditing && editingField === 'name' ? (
          <TextInput value={param.name} onChange={onNameChange} placeholder="name" focus={true} />
        ) : (
          <Text color={isSelected ? 'cyan' : undefined}>{param.name || '-'}</Text>
        )}
      </Box>

      {/* Value - TextInput ONLY when editing value */}
      <Box width={valueWidth} flexShrink={0}>
        {isEditing && editingField === 'value' ? (
          <TextInput value={param.value} onChange={onValueChange} placeholder="value" focus={true} />
        ) : (
          <Text color={isSelected ? 'cyan' : 'green'}>{param.value || '-'}</Text>
        )}
      </Box>

      {/* Type */}
      <Box width={typeWidth} flexShrink={0}>
        <Text color={isSelected ? 'yellow' : undefined}>{param.in}</Text>
        {isSelected && <Text dimColor> (c)</Text>}
      </Box>

      {/* Description */}
      <Box flexGrow={1}>
        <Text dimColor>(custom)</Text>
      </Box>
    </Box>
  );
}

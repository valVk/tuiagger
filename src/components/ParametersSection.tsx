import React, { useMemo } from 'react';
import { Box, Text } from 'ink';
import type { ParameterObject, CustomParameter } from '../types/index.js';
import { useParamNavigation } from '../hooks/useParamNavigation.js';
import type { RowItem } from '../hooks/useParamNavigation.js';
import { SpecParamRow } from './SpecParamRow.js';
import { CustomParamRow } from './CustomParamRow.js';
import { AddNewParamRow } from './AddNewParamRow.js';

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
  const rows = useMemo((): RowItem[] => {
    const items: RowItem[] = [];
    const requiredParams = parameters.filter(p => p.required);
    const optionalParams = parameters.filter(p => !p.required);
    for (const param of [...requiredParams, ...optionalParams]) {
      items.push({ type: 'spec', specParam: param });
    }
    for (const param of customParams) {
      items.push({ type: 'custom', customParam: param });
    }
    if (isTryItMode) {
      items.push({ type: 'addNew' });
    }
    return items;
  }, [parameters, customParams, isTryItMode]);

  const nav = useParamNavigation({
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
  });

  const nameWidth = 25;
  const valueWidth = 25;
  const typeWidth = 12;

  const currentRow = rows[nav.safeSelectedRow] || null;
  const isAddNewSelected = isActive && currentRow?.type === 'addNew';
  const currentIsEnum = nav.insertMode
    && currentRow?.type === 'spec'
    && !!(currentRow.specParam?.schema?.enum as string[] | undefined)?.length;

  return (
    <Box flexDirection="column" width="100%">
      <Box>
        <Text bold>PARAMETERS</Text>
        {isTryItMode && isActive && (
          <Text dimColor>
            {nav.insertMode
              ? currentIsEnum
                ? ' [EDIT] ←/→: cycle | Enter/Esc: done'
                : ' [EDIT] Tab: next field | Esc: done'
              : ' j/k: move | i: edit | d: toggle | x: del | c: type'}
          </Text>
        )}
      </Box>

      <Box width="100%">
        <Box width={3} flexShrink={0} />
        <Box width={nameWidth} flexShrink={0}><Text dimColor bold>NAME</Text></Box>
        <Box width={valueWidth} flexShrink={0}><Text dimColor bold>VALUE</Text></Box>
        <Box width={typeWidth} flexShrink={0}><Text dimColor bold>TYPE</Text></Box>
        <Box flexGrow={1}><Text dimColor bold>DESCRIPTION</Text></Box>
      </Box>

      <Box borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" />

      {rows.map((row, rowIndex) => {
        const isRowSelected = isTryItMode && isActive && nav.safeSelectedRow === rowIndex;
        const isRowEditing = isActive && nav.insertMode && nav.safeSelectedRow === rowIndex;

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
              editingField={nav.editingField}
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
            <AddNewParamRow
              key="addNew"
              isSelected={isAddNewSelected}
              addingParam={nav.addingParam}
              insertMode={nav.insertMode}
              editingNewField={nav.editingNewField}
              newParamName={nav.newParamName}
              newParamValue={nav.newParamValue}
              newParamType={nav.newParamType}
              onNameChange={nav.setNewParamName}
              onValueChange={nav.setNewParamValue}
              nameWidth={nameWidth}
              valueWidth={valueWidth}
              typeWidth={typeWidth}
            />
          );
        }

        return null;
      })}
    </Box>
  );
}

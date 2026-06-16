import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';

export interface AddNewParamRowProps {
  isSelected: boolean;
  addingParam: boolean;
  insertMode: boolean;
  editingNewField: 'name' | 'value';
  newParamName: string;
  newParamValue: string;
  newParamType: 'query' | 'path';
  onNameChange: (v: string) => void;
  onValueChange: (v: string) => void;
  nameWidth: number;
  valueWidth: number;
  typeWidth: number;
}

export function AddNewParamRow({
  isSelected,
  addingParam,
  insertMode,
  editingNewField,
  newParamName,
  newParamValue,
  newParamType,
  onNameChange,
  onValueChange,
  nameWidth,
  valueWidth,
  typeWidth,
}: AddNewParamRowProps) {
  return (
    <Box width="100%">
      <Box width={3} flexShrink={0}>
        <Text color={isSelected ? 'cyan' : undefined} dimColor={!isSelected}>{isSelected ? '>' : ' '}</Text>
      </Box>
      <Box width={nameWidth} flexShrink={0}>
        {addingParam && insertMode && editingNewField === 'name' ? (
          <TextInput value={newParamName} onChange={onNameChange} placeholder="param name" focus={true} />
        ) : addingParam && insertMode ? (
          <Text color="cyan">{newParamName || '-'}</Text>
        ) : (
          <Text color={isSelected ? 'cyan' : undefined} dimColor={!isSelected}>
            {isSelected ? '[ i: add parameter ]' : '[ + ]'}
          </Text>
        )}
      </Box>
      <Box width={valueWidth} flexShrink={0}>
        {addingParam && insertMode && editingNewField === 'value' ? (
          <TextInput value={newParamValue} onChange={onValueChange} placeholder="value" focus={true} />
        ) : addingParam && insertMode ? (
          <Text dimColor>{newParamValue || '-'}</Text>
        ) : isSelected ? (
          <Text dimColor>c: type</Text>
        ) : null}
      </Box>
      {((addingParam && insertMode) || isSelected) && (
        <Box width={typeWidth} flexShrink={0}>
          <Text color="yellow">{newParamType}</Text>
        </Box>
      )}
    </Box>
  );
}

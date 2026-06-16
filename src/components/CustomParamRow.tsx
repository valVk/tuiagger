import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import type { CustomParameter } from '../types/index.js';

export interface CustomParamRowProps {
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

export function CustomParamRow({
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
      <Box width={3} flexShrink={0}>
        <Text color="cyan">{isSelected ? '>' : ' '}</Text>
      </Box>

      <Box width={nameWidth} flexShrink={0}>
        {isEditing && editingField === 'name' ? (
          <TextInput value={param.name} onChange={onNameChange} placeholder="name" focus={true} />
        ) : (
          <Text color={isSelected ? 'cyan' : undefined}>{param.name || '-'}</Text>
        )}
      </Box>

      <Box width={valueWidth} flexShrink={0}>
        {isEditing && editingField === 'value' ? (
          <TextInput value={param.value} onChange={onValueChange} placeholder="value" focus={true} />
        ) : (
          <Text color={isSelected ? 'cyan' : 'green'}>{param.value || '-'}</Text>
        )}
      </Box>

      <Box width={typeWidth} flexShrink={0}>
        <Text color={isSelected ? 'yellow' : undefined}>{param.in}</Text>
        {isSelected && <Text dimColor> (c)</Text>}
      </Box>

      <Box flexGrow={1}>
        <Text dimColor>(custom)</Text>
      </Box>
    </Box>
  );
}

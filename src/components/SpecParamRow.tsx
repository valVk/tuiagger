import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import type { ParameterObject } from '../types/index.js';

export interface SpecParamRowProps {
  param: ParameterObject;
  value: string;
  isSelected: boolean;
  isEditing: boolean;
  isDisabled: boolean;
  onChange?: (name: string, value: string) => void;
  nameWidth: number;
  valueWidth: number;
  typeWidth: number;
}

export function SpecParamRow({
  param,
  value,
  isSelected,
  isEditing,
  isDisabled,
  onChange,
  nameWidth,
  valueWidth,
  typeWidth,
}: SpecParamRowProps) {
  const placeholder = param.schema?.default?.toString() || param.example?.toString() || '';
  const paramType = param.schema?.type || 'string';
  const enumValues = param.schema?.enum;

  const truncate = (text: string, maxLen: number) => {
    if (text.length <= maxLen - 1) return text;
    return text.slice(0, maxLen - 2) + '...';
  };

  return (
    <Box flexDirection="column" width="100%">
      <Box width="100%">
        <Box width={3} flexShrink={0}>
          {isSelected
            ? <Text color="cyan">{'> '}</Text>
            : param.required
            ? <Text color="red" dimColor>{'* '}</Text>
            : <Text>{'  '}</Text>
          }
        </Box>

        <Box width={nameWidth} flexShrink={0}>
          <Text
            color={isSelected ? 'cyan' : isDisabled ? 'gray' : undefined}
            strikethrough={isDisabled}
          >
            {truncate(param.name, nameWidth)}
          </Text>
        </Box>

        <Box width={valueWidth} flexShrink={0}>
          {isEditing && !isDisabled && enumValues?.length ? (
            <Text>
              <Text dimColor>{'< '}</Text>
              <Text color="cyan">{value || String(enumValues[0])}</Text>
              <Text dimColor>{' >'}</Text>
            </Text>
          ) : isEditing && !isDisabled && onChange ? (
            <TextInput
              value={value}
              onChange={(v) => onChange(param.name, v)}
              placeholder={placeholder || '...'}
              focus={true}
            />
          ) : (
            <Text color={isDisabled ? 'gray' : isSelected ? 'cyan' : 'green'}>
              {truncate(value || placeholder || '-', valueWidth)}
            </Text>
          )}
        </Box>

        <Box width={typeWidth} flexShrink={0} flexDirection="column">
          <Text dimColor>{param.in}</Text>
          <Text color={isDisabled ? 'gray' : 'yellow'}>{paramType}</Text>
        </Box>

        <Box flexGrow={1}>
          <Text dimColor wrap="truncate">{param.description || ''}</Text>
        </Box>
      </Box>

      {enumValues && !isDisabled && (
        <Box paddingLeft={3 + nameWidth}>
          <Text color="cyan" dimColor>Allowed: {enumValues.join(' | ')}</Text>
        </Box>
      )}
    </Box>
  );
}

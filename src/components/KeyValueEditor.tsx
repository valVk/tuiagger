import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import type { KeyValuePair } from '../types/index.js';

interface KeyValueEditorProps {
  label: string;
  pairs: KeyValuePair[];
  onChange: (pairs: KeyValuePair[]) => void;
  focusedIndex: number;
  focusedField: 'key' | 'value';
  isActive: boolean;
}

export function KeyValueEditor({
  label,
  pairs,
  onChange,
  focusedIndex,
  focusedField,
  isActive,
}: KeyValueEditorProps) {
  const updatePair = (id: string, field: 'key' | 'value', newValue: string) => {
    onChange(
      pairs.map(p => (p.id === id ? { ...p, [field]: newValue } : p))
    );
  };

  const addPair = () => {
    onChange([
      ...pairs,
      { id: Date.now().toString(), key: '', value: '', enabled: true },
    ]);
  };

  const removePair = (id: string) => {
    if (pairs.length > 1) {
      onChange(pairs.filter(p => p.id !== id));
    }
  };

  return (
    <Box flexDirection="column">
      <Box>
        <Text bold>{label}</Text>
        <Box marginLeft={2}>
          <Text dimColor>[ a: add | d: delete ]</Text>
        </Box>
      </Box>

      <Box
        flexDirection="column"
        borderStyle="single"
        borderColor="gray"
        marginTop={1}
        paddingX={1}
      >
        {/* Header */}
        <Box>
          <Box width="45%">
            <Text bold>Key</Text>
          </Box>
          <Box width="45%">
            <Text bold>Value</Text>
          </Box>
          <Box width="10%">
            <Text bold></Text>
          </Box>
        </Box>

        {/* Rows */}
        {pairs.map((pair, index) => {
          const isRowFocused = isActive && index === focusedIndex;
          const isKeyFocused = isRowFocused && focusedField === 'key';
          const isValueFocused = isRowFocused && focusedField === 'value';

          return (
            <Box key={pair.id} marginTop={1}>
              <Box width="45%">
                {isKeyFocused ? (
                  <TextInput
                    value={pair.key}
                    onChange={val => updatePair(pair.id, 'key', val)}
                    placeholder="key"
                  />
                ) : (
                  <Text color={isRowFocused ? 'cyan' : undefined}>
                    {pair.key || <Text dimColor>key</Text>}
                  </Text>
                )}
              </Box>
              <Box width="45%">
                {isValueFocused ? (
                  <TextInput
                    value={pair.value}
                    onChange={val => updatePair(pair.id, 'value', val)}
                    placeholder="value"
                  />
                ) : (
                  <Text color={isRowFocused ? 'cyan' : undefined}>
                    {pair.value || <Text dimColor>value</Text>}
                  </Text>
                )}
              </Box>
              <Box width="10%">
                <Text dimColor>[x]</Text>
              </Box>
            </Box>
          );
        })}
      </Box>
    </Box>
  );
}

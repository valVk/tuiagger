import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import type { SecuritySchemeObject } from '../types/index.js';

export interface AuthSectionProps {
  securitySchemes: [string, SecuritySchemeObject][];
  credentials: Record<string, string>;
  authCursor: number;
  editingScheme: string | null;
  editValue: string;
  isActive: boolean;
  onEditValueChange: (v: string) => void;
}

export function AuthSection({
  securitySchemes,
  credentials,
  authCursor,
  editingScheme,
  editValue,
  isActive,
  onEditValueChange,
}: AuthSectionProps) {
  if (securitySchemes.length === 0) return null;

  return (
    <>
      <Box paddingX={1} flexShrink={0} marginTop={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
        <Text bold color={isActive ? 'cyan' : undefined}>AUTH  </Text>
        {isActive && <Text dimColor>Tab: switch  j/k: move  Enter: edit  Esc: close</Text>}
      </Box>
      {securitySchemes.map(([name, scheme], i) => {
        const isCur = isActive && i === authCursor;
        const val = credentials[name] || '';
        const isEditing = editingScheme === name;
        let schemeLabel: string = scheme.type;
        if (scheme.type === 'http') schemeLabel = scheme.scheme ? `${scheme.scheme}${scheme.bearerFormat ? ` (${scheme.bearerFormat})` : ''}` : 'http';
        else if (scheme.type === 'apiKey') schemeLabel = `apiKey in ${scheme.in} as ${scheme.name}`;
        return (
          <Box key={name} paddingX={1} flexShrink={0}>
            <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
            <Text color={isCur ? 'cyan' : undefined} bold>{name}</Text>
            <Text dimColor>  {schemeLabel}  </Text>
            {isEditing
              ? <TextInput value={editValue} onChange={onEditValueChange} focus={true} />
              : val
              ? <Text color="green">{val.length > 20 ? val.slice(0, 20) + '…' : val}</Text>
              : <Text dimColor>not set</Text>
            }
          </Box>
        );
      })}
    </>
  );
}

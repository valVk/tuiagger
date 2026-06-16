import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import { useHeadersNavigation } from '../hooks/useHeadersNavigation.js';
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
  const nav = useHeadersNavigation({ headers, isActive, onChange, onTabOut, onTabBack, onInsertModeChange });

  const rows = headers.length;
  const isCursorOnAdd = isActive && nav.cursor === rows;

  return (
    <Box flexDirection="column" width="100%">
      <Box>
        <Text bold>HEADERS</Text>
        {isActive && !nav.insertMode && (
          <Text dimColor> j/k: move | i: edit | d: toggle | x: del</Text>
        )}
        {isActive && nav.insertMode && (
          <Text dimColor> Tab: switch field | Enter: confirm | Esc: cancel</Text>
        )}
      </Box>

      <Box>
        <Box width={CURSOR_WIDTH} flexShrink={0} />
        <Box width={NAME_WIDTH} flexShrink={0}><Text bold dimColor>NAME</Text></Box>
        <Box width={VALUE_WIDTH} flexShrink={0}><Text bold dimColor>VALUE</Text></Box>
      </Box>

      {headers.map((h, i) => {
        const isCur = isActive && i === nav.cursor;
        const isEditingThis = nav.insertMode && nav.editingExisting && i === nav.cursor;
        return (
          <Box key={h.id}>
            <Box width={CURSOR_WIDTH} flexShrink={0}>
              <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
            </Box>
            <Box width={NAME_WIDTH} flexShrink={0}>
              {isEditingThis && nav.editingField === 'name'
                ? <TextInput value={nav.editName} onChange={nav.setEditName} focus={true} />
                : <Text color={isCur ? 'cyan' : undefined} dimColor={!h.enabled}>{h.name || '-'}</Text>
              }
            </Box>
            <Box width={VALUE_WIDTH} flexShrink={0}>
              {isEditingThis && nav.editingField === 'value'
                ? <TextInput value={nav.editValue} onChange={nav.setEditValue} focus={true} />
                : <Text color="green" dimColor={!h.enabled}>{h.value || '-'}</Text>
              }
            </Box>
          </Box>
        );
      })}

      {(() => {
        const isAddingNew = nav.insertMode && !nav.editingExisting;
        return (
          <Box>
            <Box width={CURSOR_WIDTH} flexShrink={0}>
              <Text color={isCursorOnAdd ? 'cyan' : 'gray'}>{isCursorOnAdd ? '> ' : '  '}</Text>
            </Box>
            {isAddingNew ? (
              <>
                <Box width={NAME_WIDTH} flexShrink={0}>
                  {nav.editingField === 'name'
                    ? <TextInput value={nav.newName} onChange={nav.setNewName} focus={true} placeholder="header-name" />
                    : <Text color="cyan">{nav.newName || '-'}</Text>
                  }
                </Box>
                <Box width={VALUE_WIDTH} flexShrink={0}>
                  {nav.editingField === 'value'
                    ? <TextInput value={nav.newValue} onChange={nav.setNewValue} focus={true} placeholder="value" />
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

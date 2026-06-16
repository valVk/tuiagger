import React from 'react';
import { Box, Text } from 'ink';
import { useScreenSize } from 'fullscreen-ink';

interface StatusBarProps {
  mode: 'browse' | 'tryit' | 'manual';
  activePanel?: 'left' | 'right';
  filter?: string;
  position?: string;
}

// Always visible regardless of mode/panel
const STATIC_SHORTCUTS = [
  { key: 'q', label: 'quit' },
  { key: 'i', label: 'info' },
  { key: '?', label: 'help' },
  { key: 'Ctrl+r', label: 'reload' },
];

function getDynamicShortcuts(mode: 'browse' | 'tryit' | 'manual', activePanel?: 'left' | 'right') {
  switch (mode) {
    case 'browse':
      if (activePanel === 'left') return [
        { key: 'h/l', label: 'panels' },
        { key: 'j/k', label: 'navigate' },
        { key: 'Enter', label: 'expand tag' },
        { key: '[', label: 'wide' },
        { key: 't', label: 'try it' },
        { key: 'm', label: 'manual' },
      ];
      return [
        { key: 'h/l', label: 'panels' },
        { key: 'j/k', label: 'scroll' },
        { key: '[', label: 'wide' },
        { key: 't', label: 'try it' },
        { key: 'm', label: 'manual' },
      ];
    case 'tryit':
      return [
        { key: 'j/k', label: 'navigate' },
        { key: 'i', label: 'edit' },
        { key: 'Esc', label: 'done/cancel' },
        { key: 'e', label: 'execute' },
        { key: 'm', label: 'method' },
        { key: 'p', label: 'path' },
        { key: 'r', label: 'reset' },
      ];
    case 'manual':
      return [
        { key: 'Tab', label: 'next field' },
        { key: 'e', label: 'execute' },
        { key: 's', label: 'save' },
        { key: 'a', label: 'add row' },
        { key: 'Esc', label: 'close' },
      ];
    default:
      return [];
  }
}

function ShortcutList({ items }: { items: { key: string; label: string }[] }) {
  return (
    <>
      {items.map(({ key, label }, i) => (
        <Text key={key}>
          {i > 0 && <Text dimColor>  </Text>}
          <Text color="cyan">{key}</Text>
          <Text dimColor>:{label}</Text>
        </Text>
      ))}
    </>
  );
}

export function StatusBar({ mode, activePanel, filter, position }: StatusBarProps) {
  const { width: cols, height: rows } = useScreenSize();
  const dynamic = getDynamicShortcuts(mode, activePanel);

  return (
    <Box borderStyle="single" borderColor="gray" paddingX={1} justifyContent="space-between">
      {/* Static — fixed left, never shifts */}
      <Box>
        <ShortcutList items={STATIC_SHORTCUTS} />
      </Box>

      {/* Dynamic — context hints on right */}
      <Box>
        <ShortcutList items={dynamic} />
        {position && <Text dimColor>  {position}</Text>}
        <Text dimColor>  {cols}x{rows}</Text>
      </Box>
    </Box>
  );
}

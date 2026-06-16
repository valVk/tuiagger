import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';

interface HelpPopupProps {
  onClose: () => void;
  height: number;
}

interface KeyEntry {
  keys: string;
  desc: string;
}

interface Section {
  title: string;
  entries: KeyEntry[];
}

type Line =
  | { type: 'header'; title: string }
  | { type: 'entry'; keys: string; desc: string }
  | { type: 'spacer' };

function flattenSections(sections: Section[]): Line[] {
  const lines: Line[] = [];
  for (const s of sections) {
    lines.push({ type: 'header', title: s.title });
    for (const e of s.entries) lines.push({ type: 'entry', keys: e.keys, desc: e.desc });
    lines.push({ type: 'spacer' });
  }
  return lines;
}

const SECTIONS: Section[] = [
  {
    title: 'GLOBAL',
    entries: [
      { keys: 'q',      desc: 'Quit' },
      { keys: 'Ctrl+R', desc: 'Reload spec' },
      { keys: 'i',      desc: 'Info panel (servers / auth / envs)' },
      { keys: '[',      desc: 'Toggle left panel width' },
      { keys: '?',      desc: 'Toggle this help' },
    ],
  },
  {
    title: 'NAVIGATION',
    entries: [
      { keys: 'h / ←', desc: 'Focus left panel' },
      { keys: 'l / →', desc: 'Focus right panel' },
    ],
  },
  {
    title: 'LEFT PANEL',
    entries: [
      { keys: 'j / k',  desc: 'Move up / down' },
      { keys: 'Enter',  desc: 'Expand / collapse tag' },
      { keys: 'g / G',  desc: 'First / last item' },
      { keys: 'c / x',  desc: 'Collapse / expand all tags' },
    ],
  },
  {
    title: 'RIGHT PANEL (browse)',
    entries: [
      { keys: 'j / k', desc: 'Scroll up / down' },
      { keys: 'g',     desc: 'Scroll to top' },
      { keys: 't',     desc: 'Enter try-it-out mode' },
      { keys: 'e',     desc: 'Quick execute (uses saved params)' },
      { keys: 'm',     desc: 'New manual request' },
      { keys: 'E',     desc: 'Edit saved request' },
      { keys: 'D',     desc: 'Delete saved request' },
      { keys: '\\',    desc: 'Toggle request / response tab' },
      { keys: '/',     desc: 'Cycle response status tabs' },
    ],
  },
  {
    title: 'TRY IT OUT / MANUAL',
    entries: [
      { keys: 'e',   desc: 'Execute request' },
      { keys: 'p',   desc: 'Edit path' },
      { keys: 'm',   desc: 'Cycle HTTP method' },
      { keys: 's',   desc: 'Save request (manual mode)' },
      { keys: 'd',   desc: 'Delete request (manual edit)' },
      { keys: 'r',   desc: 'Reset overrides (tryit only)' },
      { keys: 'Esc', desc: 'Exit (NORMAL mode only)' },
    ],
  },
  {
    title: 'PARAMETERS / HEADERS',
    entries: [
      { keys: 'j / k',  desc: 'Navigate rows' },
      { keys: 'i',      desc: 'Edit value' },
      { keys: '← / →',  desc: 'Cycle enum values' },
      { keys: 'd',      desc: 'Toggle enable / disable' },
      { keys: 'x',      desc: 'Delete custom param / header' },
      { keys: 'c',      desc: 'Cycle param type (query / path)' },
      { keys: 'Tab',    desc: 'Move to next section' },
    ],
  },
  {
    title: 'BODY',
    entries: [
      { keys: 'j',   desc: 'Focus body' },
      { keys: 'i',   desc: 'Edit body (scaffolds JSON if empty)' },
      { keys: 'Esc', desc: 'Finish editing / unfocus' },
    ],
  },
  {
    title: 'RESPONSE BODY',
    entries: [
      { keys: 'J / K',  desc: 'Scroll down / up' },
      { keys: 'g / G',  desc: 'Jump to top / bottom' },
      { keys: 'v',      desc: 'Toggle visual selection' },
      { keys: 'y',      desc: 'Yank selection (or full body)' },
      { keys: 'Esc',    desc: 'Cancel visual mode' },
    ],
  },
  {
    title: 'INFO PANEL  (i)',
    entries: [
      { keys: 'Tab',   desc: 'Switch section (Servers / Auth / Envs)' },
      { keys: 'j / k', desc: 'Navigate items' },
      { keys: 'Enter', desc: 'Select server / activate env' },
      { keys: 'Esc',   desc: 'Close panel' },
    ],
  },
  {
    title: 'ENVIRONMENTS  (in Info Panel)',
    entries: [
      { keys: 'n',   desc: 'New environment' },
      { keys: 'e',   desc: 'Edit variables' },
      { keys: 'x',   desc: 'Delete environment' },
      { keys: 'i',   desc: 'Add / edit variable' },
      { keys: 'Esc', desc: 'Back to env list' },
    ],
  },
  {
    title: 'MANUAL REQUEST  (m)',
    entries: [
      { keys: 'Tab',   desc: 'Next field' },
      { keys: 'a',     desc: 'Add query / header row' },
      { keys: 'd',     desc: 'Delete selected row' },
      { keys: 'e',     desc: 'Execute request' },
      { keys: 's',     desc: 'Save request' },
      { keys: 'Esc',   desc: 'Close' },
    ],
  },
];

const KEY_W = 12;
const DESC_W = 40;
const COL_GAP = 4;

const mid = Math.ceil(SECTIONS.length / 2);
const LEFT_LINES  = flattenSections(SECTIONS.slice(0, mid));
const RIGHT_LINES = flattenSections(SECTIONS.slice(mid));
const CONTENT_H   = Math.max(LEFT_LINES.length, RIGHT_LINES.length);

function renderLine(line: Line, key: string) {
  if (line.type === 'spacer') {
    return <Box key={key} height={1} flexShrink={0} />;
  }
  if (line.type === 'header') {
    return (
      <Box key={key} width={KEY_W + DESC_W} flexShrink={0}>
        <Text bold color="cyan" wrap="truncate">{line.title}</Text>
      </Box>
    );
  }
  return (
    <Box key={key} flexShrink={0}>
      <Box width={KEY_W} flexShrink={0}>
        <Text color="yellow" wrap="truncate">{line.keys}</Text>
      </Box>
      <Box width={DESC_W} flexShrink={0}>
        <Text dimColor wrap="truncate">{line.desc}</Text>
      </Box>
    </Box>
  );
}

export function HelpPopup({ onClose, height }: HelpPopupProps) {
  const [scroll, setScroll] = useState(0);

  // border top + bottom = 2, title row = 1
  const viewHeight = height - 3;
  const maxScroll  = Math.max(0, CONTENT_H - viewHeight);

  useInput((input, key) => {
    if (input === '?' || key.escape)    { onClose(); return; }
    if (input === 'j' || key.downArrow) { setScroll(s => Math.min(s + 1, maxScroll)); return; }
    if (input === 'k' || key.upArrow)   { setScroll(s => Math.max(s - 1, 0)); return; }
    if (input === 'G')                  { setScroll(maxScroll); return; }
    if (input === 'g')                  { setScroll(0); return; }
  });

  const leftSlice  = LEFT_LINES.slice(scroll, scroll + viewHeight);
  const rightSlice = RIGHT_LINES.slice(scroll, scroll + viewHeight);

  return (
    <Box flexDirection="column" borderStyle="double" borderColor="cyan" height={height} flexGrow={1}>
      <Box paddingX={1} justifyContent="space-between" flexShrink={0}>
        <Text bold>KEYBOARD SHORTCUTS</Text>
        <Text dimColor>
          {maxScroll > 0 ? 'j/k: scroll  g/G: top/bottom  ' : ''}?/Esc: close
        </Text>
      </Box>
      <Box paddingX={1} flexShrink={0}>
        <Box flexDirection="column" width={KEY_W + DESC_W} flexShrink={0}>
          {leftSlice.map((line, i) => renderLine(line, `L-${scroll + i}`))}
        </Box>
        <Box width={COL_GAP} flexShrink={0} />
        <Box flexDirection="column" width={KEY_W + DESC_W} flexShrink={0}>
          {rightSlice.map((line, i) => renderLine(line, `R-${scroll + i}`))}
        </Box>
      </Box>
    </Box>
  );
}

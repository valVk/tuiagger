import React from 'react';
import { Box, Text } from 'ink';
import { useScreenSize } from 'fullscreen-ink';
import type { FlatListItem } from '../hooks/usePanelNavigation.js';

interface LeftPanelProps {
  items: FlatListItem[];
  selectedIndex: number;
  expandedTags: Set<string>;
  isActive: boolean;
  tagCounts: Map<string, number>;
  height?: number;
  widthPct?: number;
  title?: string;
  hasPathMethodOverride?: (method: string, path: string) => boolean;
  hasBodyOverride?: (method: string, path: string) => boolean;
  hasParamsOverride?: (method: string, path: string) => boolean;
}

export function LeftPanel({
  items,
  selectedIndex,
  expandedTags,
  isActive,
  tagCounts,
  height = 20,
  widthPct = 30,
  title,
  hasPathMethodOverride,
  hasBodyOverride,
  hasParamsOverride,
}: LeftPanelProps) {
  const { width: terminalWidth } = useScreenSize();

  // Account for: outer borders (2) + title header (2) + scroll indicator (1)
  const visibleHeight = height - 5;
  const halfVisible = Math.floor(visibleHeight / 2);

  // Available chars for path: panel width minus borders(2), paddingX*2(2), cursor(2), method+space(7)
  const panelCols = Math.floor(terminalWidth * widthPct / 100);
  const pathAvailable = panelCols - 13; // 2 borders + 2 padding + 2 cursor + 6 method + 1 space

  let startIndex = 0;
  if (items.length > visibleHeight) {
    if (selectedIndex < halfVisible) {
      startIndex = 0;
    } else if (selectedIndex > items.length - halfVisible - 1) {
      startIndex = items.length - visibleHeight;
    } else {
      startIndex = selectedIndex - halfVisible;
    }
  }

  const visibleItems = items.slice(startIndex, startIndex + visibleHeight);

  return (
    <Box
      flexDirection="column"
      borderStyle={isActive ? 'bold' : 'single'}
      borderColor={isActive ? 'cyan' : 'gray'}
      width={panelCols}
      height={height}
      flexShrink={0}
    >
      <Box paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" justifyContent="space-between">
        {title
          ? <><Text bold color="white" wrap="truncate">{title}</Text>{isActive && <Text dimColor> (i)</Text>}</>
          : <Text bold>ENDPOINTS</Text>
        }
      </Box>

      <Box flexDirection="column" paddingX={1} flexGrow={1} overflowY="hidden">
        {visibleItems.map((item, idx) => {
          const actualIndex = startIndex + idx;
          const isSelected = actualIndex === selectedIndex;

          if (item.type === 'tag') {
            const count = tagCounts.get(item.tagName) || 0;
            const isExpanded = expandedTags.has(item.tagName);

            return (
              <Box key={item.id}>
                <Text color={isSelected ? 'cyan' : undefined} bold inverse={isSelected}>
                  {isExpanded ? '▼' : '▶'} {item.tagName}
                </Text>
                <Text dimColor> ({count})</Text>
              </Box>
            );
          }

          if (item.type === 'endpoint') {
            const { method, path } = item.endpoint!;
            const methodColor = getMethodColor(method);
            const hasOverride = !!(hasPathMethodOverride?.(method, path) || hasBodyOverride?.(method, path) || hasParamsOverride?.(method, path));
            const displayPath = truncatePath(path, Math.max(pathAvailable, 8));

            const cursorChar = hasOverride ? '~ ' : '  ';

            return (
              <Box key={item.id}>
                <Text>{cursorChar}</Text>
                <Text color={methodColor} bold>
                  {method.toUpperCase().padEnd(6)}
                </Text>
                <Text> </Text>
                <Text inverse={isSelected} color={isSelected ? 'cyan' : undefined}>
                  {displayPath}
                </Text>
              </Box>
            );
          }

          if (item.type === 'savedRequest') {
            const { method, path } = item.savedRequest!;
            const methodColor = getMethodColor(method);
            // saved: cursor(2) + "* "(2) + method+space(7) = 11 extra chars
            const maxPath = truncatePath(path, Math.max(panelCols - 15, 8));

            return (
              <Box key={item.id}>
                {isSelected && <Text color="cyan">&gt; </Text>}
                {!isSelected && <Text>  </Text>}
                <Text color="yellow">* </Text>
                <Text color={methodColor} bold>
                  {method.toUpperCase().padEnd(6)}
                </Text>
                <Text> </Text>
                <Text inverse={isSelected} color={isSelected ? 'cyan' : undefined}>
                  {maxPath}
                </Text>
              </Box>
            );
          }

          return null;
        })}

        {/* Scroll indicators */}
        {startIndex > 0 && (
          <Box position="absolute" marginTop={-visibleHeight}>
            <Text dimColor>  ↑ more</Text>
          </Box>
        )}
      </Box>

      {/* Scroll position indicator — always rendered to prevent layout shift */}
      <Box paddingX={1} justifyContent="flex-end">
        <Text dimColor>
          {items.length > visibleHeight ? `${selectedIndex + 1}/${items.length}` : ''}
        </Text>
      </Box>
    </Box>
  );
}

function getMethodColor(method: string): string {
  const colors: Record<string, string> = {
    get: 'blue',
    post: 'green',
    put: 'yellow',
    delete: 'red',
    patch: 'cyan',
    head: 'magenta',
    options: 'gray',
  };
  return colors[method.toLowerCase()] || 'white';
}

function truncatePath(path: string, maxLength: number): string {
  if (path.length <= maxLength) return path;
  return path.slice(0, maxLength - 2) + '..';
}

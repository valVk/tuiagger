import React from 'react';
import { Box, Text } from 'ink';

interface ServerEntry {
  url: string;
  description?: string;
}

export interface ServersSectionProps {
  servers: ServerEntry[];
  selectedServer: number;
  cursor: number;
  isActive: boolean;
}

export function ServersSection({ servers, selectedServer, cursor, isActive }: ServersSectionProps) {
  return (
    <>
      <Box paddingX={1} flexShrink={0} marginTop={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
        <Text bold color={isActive ? 'cyan' : undefined}>SERVERS  </Text>
        {isActive && <Text dimColor>Tab: switch  j/k: move  Enter: select  Esc: close</Text>}
      </Box>
      {servers.map((server, i) => {
        const isSelected = i === selectedServer;
        const isCur = isActive && i === cursor;
        return (
          <Box key={i} paddingX={1} flexShrink={0}>
            <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
            <Text color={isCur ? 'cyan' : undefined} bold={isSelected}>{server.url}</Text>
            {server.description && <Text dimColor>  {server.description}</Text>}
            {isSelected && <Text color="green">  active</Text>}
          </Box>
        );
      })}
    </>
  );
}

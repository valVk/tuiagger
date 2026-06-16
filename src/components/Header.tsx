import React from 'react';
import { Box, Text } from 'ink';
import type { OpenAPISpec } from '../types/index.js';

interface HeaderProps {
  spec: OpenAPISpec;
  selectedServer: number;
  onServerChange?: (index: number) => void;
  collectionName?: string;
  activeEnvName?: string;
}

export function Header({ spec, selectedServer, collectionName, activeEnvName }: HeaderProps) {
  const servers = spec.servers || [{ url: 'http://localhost', description: 'Default' }];
  const currentServer = servers[selectedServer] || servers[0];

  const serverUrl = currentServer.url.length > 50
    ? currentServer.url.slice(0, 48) + '..'
    : currentServer.url;

  return (
    <Box flexDirection="row" paddingX={1} borderStyle="single" borderColor="gray" justifyContent="space-between">
      <Box flexShrink={0}>
        {collectionName && (
          <Text color="yellow">[{collectionName}] </Text>
        )}
        <Text bold color="white">{spec.info.title}</Text>
        <Text dimColor> v{spec.info.version}</Text>
      </Box>
      <Box flexShrink={0}>
        {activeEnvName && <Text color="yellow">env: {activeEnvName}  </Text>}
        <Text dimColor>server: </Text>
        <Text color="cyan">{serverUrl}</Text>
      </Box>
    </Box>
  );
}

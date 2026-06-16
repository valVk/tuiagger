import { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { getStatusColor } from '../utils/colors.js';
import { formatSchema } from '../utils/parser.js';
import type { ResponsesObject } from '../types/index.js';

interface ResponsesSectionProps {
  responses: ResponsesObject;
  specComponents?: Record<string, unknown>;
  isActive?: boolean;
}

export function ResponsesSection({ responses, specComponents, isActive = false }: ResponsesSectionProps) {
  const statusCodes = Object.keys(responses).sort();
  const [selectedTab, setSelectedTab] = useState(0);

  useInput(
    (input) => {
      if (input === '/') {
        setSelectedTab(prev => (prev + 1) % statusCodes.length);
      }
    },
    { isActive }
  );

  const safeTab = Math.min(selectedTab, statusCodes.length - 1);
  const activeCode = statusCodes[safeTab];
  const activeResponse = activeCode ? responses[activeCode] : null;
  const schema = activeResponse?.content ? Object.values(activeResponse.content)[0]?.schema : null;
  const schemaStr = schema ? formatSchema(schema, 0, specComponents) : null;
  const contentTypes = activeResponse?.content ? Object.keys(activeResponse.content) : [];

  return (
    <Box flexDirection="column" width="100%">
      <Box>
        <Text bold>Responses</Text>
        {isActive && statusCodes.length > 1 && <Text dimColor> /:next</Text>}
      </Box>

      {/* Tab bar */}
      <Box>
        {statusCodes.map((code, i) => {
          const color = getStatusColor(parseInt(code, 10));
          const isTab = i === safeTab;
          return (
            <Box key={code} marginRight={1}>
              <Text color={isTab ? color : undefined} inverse={isTab} bold={isTab}> {code} </Text>
            </Box>
          );
        })}
      </Box>

      {/* Active tab content */}
      {activeResponse && (
        <Box flexDirection="column" paddingLeft={1}>
          <Text dimColor>{activeResponse.description}{contentTypes.length > 0 ? ` (${contentTypes.join(', ')})` : ''}</Text>
          {schemaStr && schemaStr.split('\n').map((line, i) => (
            <Text key={i} dimColor>{line}</Text>
          ))}
        </Box>
      )}
    </Box>
  );
}

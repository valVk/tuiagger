import React from 'react';
import { Box, Text } from 'ink';
import { getMethodColor } from '../utils/colors.js';

interface MethodBadgeProps {
  method: string;
  width?: number;
}

export function MethodBadge({ method, width = 8 }: MethodBadgeProps) {
  const color = getMethodColor(method);
  const upperMethod = method.toUpperCase().padEnd(width);

  return (
    <Box>
      <Text backgroundColor={color} color="white" bold>
        {` ${upperMethod}`}
      </Text>
    </Box>
  );
}

import { useState } from 'react';
import { useInput } from 'ink';

interface UseServersKeyboardOptions {
  serversCount: number;
  initialCursor: number;
  isActive: boolean;
  onServerChange: (index: number) => void;
  onClose: () => void;
}

export function useServersKeyboard({
  serversCount,
  initialCursor,
  isActive,
  onServerChange,
  onClose,
}: UseServersKeyboardOptions) {
  const [cursor, setCursor] = useState(initialCursor);

  useInput((input, key) => {
    if (!isActive) return;
    if (input === 'j' || key.downArrow) { setCursor(p => Math.min(p + 1, serversCount - 1)); return; }
    if (input === 'k' || key.upArrow)   { setCursor(p => Math.max(p - 1, 0)); return; }
    if (key.return) { onServerChange(cursor); onClose(); return; }
  }, { isActive });

  return { cursor };
}

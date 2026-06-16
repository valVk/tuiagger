import { useState } from 'react';
import { useInput } from 'ink';
import type { SecuritySchemeObject } from '../types/index.js';

interface UseAuthKeyboardOptions {
  securitySchemes: [string, SecuritySchemeObject][];
  credentials: Record<string, string>;
  isActive: boolean;
  onSetCredential: (name: string, val: string) => void;
}

export function useAuthKeyboard({
  securitySchemes,
  credentials,
  isActive,
  onSetCredential,
}: UseAuthKeyboardOptions) {
  const [authCursor, setAuthCursor] = useState(0);
  const [editingScheme, setEditingScheme] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');

  const isInsertMode = !!editingScheme;

  useInput((input, key) => {
    if (!isActive) return;

    if (editingScheme) {
      if (key.escape || key.return) {
        onSetCredential(editingScheme, editValue);
        setEditingScheme(null);
      }
      return;
    }

    if (input === 'j' || key.downArrow) { setAuthCursor(p => Math.min(p + 1, securitySchemes.length - 1)); return; }
    if (input === 'k' || key.upArrow)   { setAuthCursor(p => Math.max(p - 1, 0)); return; }
    if (key.return && securitySchemes.length > 0) {
      const [name] = securitySchemes[authCursor];
      setEditValue(credentials[name] || '');
      setEditingScheme(name);
      return;
    }
  }, { isActive });

  return { authCursor, editingScheme, editValue, setEditValue, isInsertMode };
}

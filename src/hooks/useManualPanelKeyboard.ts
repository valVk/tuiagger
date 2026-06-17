import { useState, useEffect } from 'react';
import { useInput } from 'ink';
import type { HttpMethodType } from '../types/index.js';

const HTTP_METHODS: HttpMethodType[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

interface UseManualPanelKeyboardOptions {
  isActive: boolean;
  method: string;
  path: string;
  onMethodChange: (method: string) => void;
  onPathChange: (path: string) => void;
  onNormalModeChange?: (isNormal: boolean) => void;
}

interface UseManualPanelKeyboardResult {
  editingPath: boolean;
  bodyTabFocused: boolean;
  editingBody: boolean;
  paramInsertMode: boolean;
  headersFocused: boolean;
  headersInsertMode: boolean;
  setEditingBody: (v: boolean) => void;
  setBodyTabFocused: (v: boolean) => void;
  setParamInsertMode: (v: boolean) => void;
  setHeadersFocused: (v: boolean) => void;
  setHeadersInsertMode: (v: boolean) => void;
}

export function useManualPanelKeyboard({
  isActive,
  method,
  path,
  onMethodChange,
  onPathChange,
  onNormalModeChange,
}: UseManualPanelKeyboardOptions): UseManualPanelKeyboardResult {
  const [editingPath, setEditingPath] = useState(false);
  const [bodyTabFocused, setBodyTabFocused] = useState(false);
  const [editingBody, setEditingBody] = useState(false);
  const [paramInsertMode, setParamInsertMode] = useState(false);
  const [headersFocused, setHeadersFocused] = useState(false);
  const [headersInsertMode, setHeadersInsertMode] = useState(false);

  useEffect(() => {
    onNormalModeChange?.(!editingPath && !editingBody && !paramInsertMode && !headersInsertMode);
  }, [editingPath, editingBody, paramInsertMode, headersInsertMode]);

  useInput(
    (input, key) => {
      if (!isActive) return;

      if (editingPath) {
        if (key.escape || key.return) setEditingPath(false);
        return;
      }
      if (editingBody) {
        if (key.escape) setEditingBody(false);
        return;
      }
      if (paramInsertMode || headersInsertMode) return;
      if (headersFocused) return;

      if (bodyTabFocused) {
        if (input === 'i') { setEditingBody(true); return; }
        if (input === 'k' || key.upArrow || key.escape) { setBodyTabFocused(false); return; }
        return;
      }

      if (input === 'm') {
        const currentIndex = HTTP_METHODS.indexOf((method || 'GET').toUpperCase() as HttpMethodType);
        onMethodChange(HTTP_METHODS[(currentIndex + 1) % HTTP_METHODS.length]);
        return;
      }

      if (input === 'p') {
        setEditingPath(true);
        return;
      }
    },
    { isActive }
  );

  return {
    editingPath,
    bodyTabFocused,
    editingBody,
    paramInsertMode,
    headersFocused,
    headersInsertMode,
    setEditingBody,
    setBodyTabFocused,
    setParamInsertMode,
    setHeadersFocused,
    setHeadersInsertMode,
  };
}

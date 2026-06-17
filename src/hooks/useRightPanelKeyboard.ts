import { useState, useEffect } from 'react';
import { useInput } from 'ink';
import { scaffoldPlaceholder } from '../utils/parser.js';
import type { HttpMethodType } from '../types/index.js';
import type { FlatListItem, RightPanelMode } from './usePanelNavigation.js';

const HTTP_METHODS: HttpMethodType[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];
const LINES_PER_PARAM = 2;
const HEADER_LINES = 10;

interface UseRightPanelKeyboardOptions {
  isActive: boolean;
  mode: RightPanelMode;
  selectedItem: FlatListItem | null;
  body: string;
  overridePath: string | undefined;
  overrideMethod: string | undefined;
  showResetConfirm: boolean;
  specComponents: Record<string, unknown> | undefined;
  height: number;
  scrollOffset: number;
  onOverridePathChange?: (path: string) => void;
  onOverrideMethodChange?: (method: string) => void;
  onBodyChange: (value: string) => void;
  onResetOverrides?: () => void;
  onResetConfirmResponse?: (confirmed: boolean) => void;
  onNormalModeChange?: (isNormal: boolean) => void;
  onScrollChange?: (offset: number) => void;
  onEditingChange?: (editing: boolean) => void;
}

interface UseRightPanelKeyboardResult {
  editingPath: boolean;
  bodyTabFocused: boolean;
  editingBody: boolean;
  paramResetKey: number;
  paramInsertMode: boolean;
  headersFocused: boolean;
  headersInsertMode: boolean;
  setHeadersFocused: (v: boolean) => void;
  setParamInsertMode: (v: boolean) => void;
  setHeadersInsertMode: (v: boolean) => void;
  scrollToParamRow: (rowIndex: number) => void;
}

export function useRightPanelKeyboard({
  isActive,
  mode,
  selectedItem,
  body,
  overridePath,
  overrideMethod,
  showResetConfirm,
  specComponents,
  height,
  scrollOffset,
  onOverridePathChange,
  onOverrideMethodChange,
  onBodyChange,
  onResetOverrides,
  onResetConfirmResponse,
  onNormalModeChange,
  onScrollChange,
  onEditingChange,
}: UseRightPanelKeyboardOptions): UseRightPanelKeyboardResult {
  const [editingPath, setEditingPath] = useState(false);
  const [bodyTabFocused, setBodyTabFocused] = useState(false);
  const [editingBody, setEditingBody] = useState(false);
  const [paramResetKey, setParamResetKey] = useState(0);
  const [paramInsertMode, setParamInsertMode] = useState(false);
  const [headersFocused, setHeadersFocused] = useState(false);
  const [headersInsertMode, setHeadersInsertMode] = useState(false);

  useEffect(() => {
    onNormalModeChange?.(!editingPath && !editingBody && !paramInsertMode && !headersInsertMode);
  }, [editingPath, editingBody, paramInsertMode, headersInsertMode]);

  const setEditingBodyWithSignal = (val: boolean) => {
    setEditingBody(val);
    onEditingChange?.(val);
  };

  const scrollToParamRow = (rowIndex: number) => {
    if (!onScrollChange) return;
    const rowTop = HEADER_LINES + rowIndex * LINES_PER_PARAM;
    const rowBottom = rowTop + LINES_PER_PARAM;
    const visibleHeight = height - 4;
    if (rowTop >= scrollOffset && rowBottom <= scrollOffset + visibleHeight) return;
    if (rowTop < scrollOffset) {
      onScrollChange(Math.max(0, rowTop));
    } else {
      onScrollChange(Math.max(0, rowBottom - visibleHeight));
    }
  };

  useInput(
    (input, key) => {
      const isTryItMode = mode === 'tryItOut';
      if (!isTryItMode || !isActive) return;

      if (showResetConfirm) {
        if (input === 'y' || input === 'Y') onResetConfirmResponse?.(true);
        else if (input === 'n' || input === 'N' || key.escape) onResetConfirmResponse?.(false);
        return;
      }

      if (editingPath) {
        if (key.escape || key.return) setEditingPath(false);
        return;
      }
      if (editingBody) {
        if (key.escape) setEditingBodyWithSignal(false);
        return;
      }
      if (paramInsertMode || headersInsertMode) return;
      if (headersFocused) return;

      if (bodyTabFocused) {
        if (input === 'i') {
          if (!body && selectedItem?.type === 'endpoint') {
            const schema = Object.values(selectedItem.endpoint!.operation.requestBody?.content || {})[0]?.schema;
            if (schema) {
              const scaffold = scaffoldPlaceholder(schema, specComponents);
              if (scaffold !== null) onBodyChange(JSON.stringify(scaffold, null, 2));
            }
          }
          setEditingBodyWithSignal(true);
          return;
        }
        if (input === 'k' || key.upArrow || key.escape) { setBodyTabFocused(false); return; }
        return;
      }

      if (input === 'm' && selectedItem?.type === 'endpoint') {
        const baseMethod = selectedItem.endpoint!.method;
        const currentMethod = (overrideMethod || baseMethod).toUpperCase() as HttpMethodType;
        const currentIndex = HTTP_METHODS.indexOf(currentMethod);
        onOverrideMethodChange?.(HTTP_METHODS[(currentIndex + 1) % HTTP_METHODS.length]);
        return;
      }

      if (input === 'p' && selectedItem?.type === 'endpoint') {
        setEditingPath(true);
        if (!overridePath) onOverridePathChange?.(selectedItem.endpoint!.path);
        return;
      }

      if (input === 'r' && selectedItem?.type === 'endpoint') {
        onResetOverrides?.();
        return;
      }
    },
    { isActive: mode === 'tryItOut' && isActive }
  );

  return {
    editingPath,
    bodyTabFocused,
    editingBody,
    paramResetKey,
    paramInsertMode,
    headersFocused,
    headersInsertMode,
    setHeadersFocused,
    setParamInsertMode,
    setHeadersInsertMode,
    scrollToParamRow,
  };
}

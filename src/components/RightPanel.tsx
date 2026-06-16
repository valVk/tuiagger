import React, { useState, useEffect } from 'react';
import { Box, Newline, Text, useInput } from 'ink';
import { spawn } from 'child_process';
import TextInput from 'ink-text-input';
import { TextArea } from 'react-ink-textarea';
import type { HttpMethodType } from '../types/index.js';
import { MethodBadge } from './MethodBadge.js';
import { ParametersSection } from './ParametersSection.js';
import { HeadersSection } from './HeadersSection.js';
import { ResponsesSection } from './ResponsesSection.js';
import type { FlatListItem, RightPanelMode } from '../hooks/usePanelNavigation.js';
import type { ResponseState, CustomParameter, SavedRequest } from '../types/index.js';
import { formatSchema, scaffoldPlaceholder } from '../utils/parser.js';

interface RightPanelProps {
  selectedItem: FlatListItem | null;
  mode: RightPanelMode;
  isActive: boolean;
  scrollOffset: number;
  parameterValues: Record<string, string>;
  onParameterChange: (name: string, value: string) => void;
  body: string;
  onBodyChange: (value: string) => void;
  response?: ResponseState | null;
  curl?: string;
  isLoading?: boolean;
  height?: number;
  onScrollChange?: (offset: number) => void;
  // Custom params support
  customParams?: CustomParameter[];
  onCustomParamsChange?: (params: CustomParameter[]) => void;
  disabledParams?: string[];
  onDisabledParamsChange?: (disabled: string[]) => void;
  // Path/method override support
  overridePath?: string;
  overrideMethod?: string;
  onOverridePathChange?: (path: string) => void;
  onOverrideMethodChange?: (method: string) => void;
  onResetOverrides?: () => void;
  showResetConfirm?: boolean;
  onResetConfirmResponse?: (confirmed: boolean) => void;
  onEditingChange?: (editing: boolean) => void;
  specComponents?: Record<string, unknown>;
  manualMode?: boolean;
  editingRequest?: SavedRequest;
  onNormalModeChange?: (isNormal: boolean) => void;
}

// Flat param rows: 1 line each + 1 for optional enum row
const LINES_PER_PARAM = 2;
// Lines before first param: method/path + buttons + summary + desc + hint + header + divider + section label
const HEADER_LINES = 10;

export function RightPanel({
  selectedItem,
  mode,
  isActive,
  scrollOffset,
  parameterValues,
  onParameterChange,
  body,
  onBodyChange,
  response,
  curl,
  isLoading,
  height = 20,
  onScrollChange,
  customParams = [],
  onCustomParamsChange,
  disabledParams = [],
  onDisabledParamsChange,
  overridePath,
  overrideMethod,
  onOverridePathChange,
  onOverrideMethodChange,
  onResetOverrides,
  showResetConfirm,
  onResetConfirmResponse,
  onEditingChange,
  specComponents,
  manualMode = false,
  editingRequest,
  onNormalModeChange,
}: RightPanelProps) {
  const [editingPath, setEditingPath] = useState(false);
  const [bodyTabFocused, setBodyTabFocused] = useState(false);
  const [editingBody, setEditingBody] = useState(false);
  const [paramResetKey, setParamResetKey] = useState(0);
  const [paramInsertMode, setParamInsertMode] = useState(false);
  const [headersFocused, setHeadersFocused] = useState(false);
  const [headersInsertMode, setHeadersInsertMode] = useState(false);
  const [responseTab, setResponseTab] = useState<'response' | 'request'>('response');
  const [responseScroll, setResponseScroll] = useState(0);
  const [responseCursor, setResponseCursor] = useState(0);
  const [visualMode, setVisualMode] = useState(false);
  const [visualAnchor, setVisualAnchor] = useState(0);
  const [yankMessage, setYankMessage] = useState(false);

  const RESPONSE_VIEWPORT = 15;

  useEffect(() => {
    setResponseScroll(0);
    setResponseCursor(0);
    setVisualMode(false);
    setVisualAnchor(0);
    setYankMessage(false);
  }, [response]);

  useEffect(() => {
    onNormalModeChange?.(!editingPath && !editingBody && !paramInsertMode && !headersInsertMode);
  }, [editingPath, editingBody, paramInsertMode, headersInsertMode]);

  const setEditingBodyWithSignal = (val: boolean) => {
    setEditingBody(val);
    onEditingChange?.(val);
  };

  const HTTP_METHODS: HttpMethodType[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

  // Handle keyboard input for path/method editing
  useInput(
    (input, key) => {
      const isTryItMode = mode === 'tryItOut';
      if (!isTryItMode || !isActive) return;

      // If showing reset confirmation, handle y/n
      if (showResetConfirm) {
        if (input === 'y' || input === 'Y') {
          onResetConfirmResponse?.(true);
        } else if (input === 'n' || input === 'N' || key.escape) {
          onResetConfirmResponse?.(false);
        }
        return;
      }

      // If editing path, only handle escape
      if (editingPath) {
        if (key.escape || key.return) {
          setEditingPath(false);
        }
        return;
      }

      // If editing body, only handle escape
      if (editingBody) {
        if (key.escape) {
          setEditingBodyWithSignal(false);
        }
        return;
      }

      // If param or headers section is in insert mode, block all panel-level shortcuts
      if (paramInsertMode || headersInsertMode) return;

      // If headers section is focused, block panel shortcuts
      if (headersFocused) return;

      // If body is tab-focused (nav mode on body)
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
        if (input === 'k' || key.upArrow) {
          setBodyTabFocused(false);
          return;
        }
        if (key.escape) {
          setBodyTabFocused(false);
          return;
        }
        return;
      }

      // 'm' to cycle method
      if (input === 'm' && (selectedItem?.type === 'endpoint' || manualMode)) {
        const baseMethod = manualMode ? 'GET' : selectedItem!.endpoint!.method;
        const currentMethod = (overrideMethod || baseMethod).toUpperCase() as HttpMethodType;
        const currentIndex = HTTP_METHODS.indexOf(currentMethod);
        const nextIndex = (currentIndex + 1) % HTTP_METHODS.length;
        onOverrideMethodChange?.(HTTP_METHODS[nextIndex]);
        return;
      }

      // 'p' to edit path
      if (input === 'p' && (selectedItem?.type === 'endpoint' || manualMode)) {
        setEditingPath(true);
        if (!overridePath && !manualMode) {
          onOverridePathChange?.(selectedItem!.endpoint!.path);
        }
        return;
      }

      // 'r' to reset (spec endpoints only)
      if (input === 'r' && selectedItem?.type === 'endpoint') {
        onResetOverrides?.();
        return;
      }
    },
    { isActive: mode === 'tryItOut' && isActive }
  );

  useInput(
    (input, key) => {
      if (input === '\\' && response) {
        setResponseTab(prev => prev === 'response' ? 'request' : 'response');
        onScrollChange?.(0);
        return;
      }

      if (!response || responseTab !== 'response') return;

      const bodyLines = response.body.split('\n');
      const maxLine = bodyLines.length - 1;

      const moveCursor = (next: number) => {
        const clamped = Math.max(0, Math.min(next, maxLine));
        setResponseCursor(clamped);
        setResponseScroll(s => {
          if (clamped < s) return clamped;
          if (clamped >= s + RESPONSE_VIEWPORT) return clamped - RESPONSE_VIEWPORT + 1;
          return s;
        });
      };

      if (key.escape) {
        setVisualMode(false);
        return;
      }

      if (input === 'v') {
        setVisualMode(v => {
          if (!v) setVisualAnchor(responseCursor);
          return !v;
        });
        return;
      }

      if (input === 'J') {
        moveCursor(responseCursor + 1);
        return;
      }
      if (input === 'K') {
        moveCursor(responseCursor - 1);
        return;
      }
      if (input === 'G') {
        moveCursor(maxLine);
        return;
      }
      if (input === 'g') {
        moveCursor(0);
        return;
      }

      if (input === 'y') {
        let textToCopy: string;
        if (visualMode) {
          const selStart = Math.min(visualAnchor, responseCursor);
          const selEnd = Math.max(visualAnchor, responseCursor);
          textToCopy = bodyLines.slice(selStart, selEnd + 1).join('\n');
          setVisualMode(false);
        } else {
          textToCopy = response.body;
        }
        try {
          const proc = spawn('pbcopy', [], { stdio: 'pipe' });
          proc.stdin.write(textToCopy);
          proc.stdin.end();
          proc.on('error', () => {
            const proc2 = spawn('xclip', ['-selection', 'clipboard'], { stdio: 'pipe' });
            proc2.stdin.write(textToCopy);
            proc2.stdin.end();
          });
        } catch {}
        setYankMessage(true);
        setTimeout(() => setYankMessage(false), 1500);
        return;
      }
    },
    { isActive: isActive && mode !== 'tryItOut' }
  );

  const scrollToParamRow = (rowIndex: number) => {
    if (!onScrollChange) return;

    const rowTop = HEADER_LINES + rowIndex * LINES_PER_PARAM;
    const rowBottom = rowTop + LINES_PER_PARAM;
    const visibleHeight = height - 4;

    // Already fully visible — don't touch scroll
    if (rowTop >= scrollOffset && rowBottom <= scrollOffset + visibleHeight) return;

    if (rowTop < scrollOffset) {
      // Row scrolled above viewport — bring it to top
      onScrollChange(Math.max(0, rowTop));
    } else {
      // Row below viewport — scroll down just enough to show it
      onScrollChange(Math.max(0, rowBottom - visibleHeight));
    }
  };

  const handleParamFocusChange = (paramIndex: number) => {
    scrollToParamRow(paramIndex);
  };

  const handleBodyFocus = (totalParamRows: number) => {
    scrollToParamRow(totalParamRows);
  };
  const renderManualTryit = () => {
    const displayMethod = overrideMethod || 'GET';
    const displayPath = overridePath ?? '';
    const nonHeaderCustomParams = customParams.filter(p => p.in !== 'header');
    const headerCustomParams = customParams.filter(p => p.in === 'header');

    const handleHeadersChange = (updated: CustomParameter[]) => {
      onCustomParamsChange?.([...nonHeaderCustomParams, ...updated]);
    };

    return (
      <Box flexDirection="column">
        {/* Header badge + method + path */}
        <Box paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="cyan" width="100%" flexShrink={0}>
          <Text bold backgroundColor="cyan" color="black">
            {editingRequest ? ` EDITING: ${editingRequest.name} ` : ' MANUAL REQUEST '}
          </Text>
          <Text> </Text>
          <Box>
            <MethodBadge method={displayMethod} />
            <Text dimColor> (m)</Text>
          </Box>
          <Text> </Text>
          {editingPath ? (
            <Box borderStyle="single" borderColor="green" flexGrow={1}>
              <TextInput
                value={displayPath}
                onChange={(v) => onOverridePathChange?.(v)}
                focus={true}
              />
            </Box>
          ) : (
            <Box flexGrow={1}>
              <Text bold color={displayPath ? undefined : 'red'}>
                {displayPath || '/path/required'}
              </Text>
              <Text dimColor> (p)</Text>
            </Box>
          )}
        </Box>

        {/* Action bar */}
        {isActive && (
          <Box justifyContent="flex-end" flexShrink={0}>
            <Text dimColor>[ </Text>
            <Text color="green" bold>Execute (e)</Text>
            <Text dimColor> ][ </Text>
            <Text color="cyan">Save (s)</Text>
            <Text dimColor> ]</Text>
            {editingRequest && (
              <>
                <Text dimColor>[ </Text>
                <Text color="red">Delete (d)</Text>
                <Text dimColor> ]</Text>
              </>
            )}
            <Text dimColor>[ Cancel (Esc) ] </Text>
          </Box>
        )}

        <Box flexDirection="column" paddingX={1} width="100%" marginTop={1}>
          <Box flexShrink={0} flexDirection="column" width="100%">
            <HeadersSection
              headers={headerCustomParams}
              onChange={handleHeadersChange}
              isActive={isActive && headersFocused}
              onTabOut={() => setHeadersFocused(false)}
              onTabBack={() => setHeadersFocused(false)}
              onInsertModeChange={setHeadersInsertMode}
            />
          </Box>
          <Box marginTop={1} flexDirection="column" width="100%" flexShrink={0}>
            <ParametersSection
              parameters={[]}
              isTryItMode={true}
              values={{}}
              onChange={() => {}}
              isActive={isActive && !bodyTabFocused && !editingBody && !headersFocused}
              onFocusChange={handleParamFocusChange}
              onTabOut={['POST', 'PUT', 'PATCH', 'DELETE'].includes(displayMethod.toUpperCase()) ? () => setBodyTabFocused(true) : undefined}
              onTabBack={() => setHeadersFocused(true)}
              resetKey={paramResetKey}
              customParams={nonHeaderCustomParams}
              onCustomParamsChange={(updated) => onCustomParamsChange?.([...updated, ...headerCustomParams])}
              disabledParams={[]}
              onDisabledParamsChange={() => {}}
              onInsertModeChange={setParamInsertMode}
            />
          </Box>

          {/* Body editor — only for methods that support a body */}
          {['POST', 'PUT', 'PATCH', 'DELETE'].includes(displayMethod.toUpperCase()) && (
            <Box flexDirection="column" flexShrink={0} marginTop={1}>
              <Box flexShrink={0}>
                <Text bold>BODY </Text>
                <Text dimColor>application/json</Text>
              </Box>
              <Box borderStyle="round" borderColor={editingBody ? 'green' : bodyTabFocused ? 'cyan' : 'gray'} paddingX={1} flexDirection="column">
                {!editingBody && !body ? (
                  <Text dimColor>{bodyTabFocused ? 'i: edit  k: back' : 'j: focus  i: edit'}</Text>
                ) : (
                  <>
                    <TextArea
                      value={body}
                      onChange={onBodyChange}
                      onSubmit={() => setEditingBodyWithSignal(false)}
                      focus={editingBody}
                      viewportLines={10}
                      onTab={(shift) => { if (!shift) onBodyChange(body + '  '); }}
                      onFirstLineUp={() => setEditingBodyWithSignal(false)}
                    />
                    {editingBody && (
                      <Text dimColor>Enter: done  Shift+Enter: newline  Esc: cancel</Text>
                    )}
                  </>
                )}
              </Box>
            </Box>
          )}

          {/* Response */}
          {response && (
            <Box marginTop={1} flexShrink={0}>
              {renderResponse()}
            </Box>
          )}
        </Box>
      </Box>
    );
  };

  const renderContent = () => {
    if (manualMode) return renderManualTryit();

    if (!selectedItem) {
      return (
        <Box flexDirection="column" paddingX={1}>
          <Text dimColor>Select an endpoint from the left panel (j/k to navigate)</Text>
        </Box>
      );
    }

    if (selectedItem.type === 'tag') {
      return (
        <Box flexDirection="column" paddingX={1}>
          <Text bold>{selectedItem.tagName}</Text>
          <Text dimColor> — press Enter to expand/collapse</Text>
        </Box>
      );
    }

    if (selectedItem.type === 'endpoint') {
      return renderEndpointDetails(selectedItem);
    }

    if (selectedItem.type === 'savedRequest') {
      return renderSavedRequestDetails(selectedItem);
    }

    return null;
  };

  const renderEndpointDetails = (item: FlatListItem) => {
    const { method, path, operation } = item.endpoint!;
    const isTryItMode = mode === 'tryItOut';

    // Split customParams: non-header → ParametersSection, header → HeadersSection
    const nonHeaderCustomParams = customParams.filter(p => p.in !== 'header');
    const headerCustomParams = customParams.filter(p => p.in === 'header');

    const handleHeadersChange = (updated: CustomParameter[]) => {
      onCustomParamsChange?.([...nonHeaderCustomParams, ...updated]);
    };

    // Use override values if set, otherwise use original
    const displayMethod = overrideMethod || method;
    const displayPath = overridePath || path;
    const isMethodModified = overrideMethod && overrideMethod !== method;
    const isPathModified = overridePath && overridePath !== path;
    const hasAnyOverride = isMethodModified || isPathModified;

    return (
      <Box flexDirection="column">
        {/* Reset confirmation */}
        {showResetConfirm && (
          <Box paddingX={1} paddingY={1} borderStyle="double" borderColor="yellow" width="100%">
            <Text color="yellow" bold>Reset all overrides for this endpoint? (y/n)</Text>
          </Box>
        )}

        {/* Header with method/path */}
        <Box paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" width="100%" flexShrink={0}>
          {isTryItMode ? (
            <>
              <Box>
                <MethodBadge method={displayMethod} />
                {isMethodModified && <Text color="yellow">*</Text>}
                {isTryItMode && <Text dimColor> (m)</Text>}
              </Box>
              <Text> </Text>
              {editingPath ? (
                <Box borderStyle="single" borderColor="green" flexGrow={1}>
                  <TextInput
                    value={displayPath}
                    onChange={(v) => onOverridePathChange?.(v)}
                    focus={true}
                  />
                </Box>
              ) : (
                <Box>
                  <Text bold color={isPathModified ? 'yellow' : undefined}>{displayPath}</Text>
                  {isPathModified && <Text color="yellow">*</Text>}
                  {isTryItMode && <Text dimColor> (p)</Text>}
                </Box>
              )}
            </>
          ) : (
            <>
              <MethodBadge method={displayMethod} />
              {isMethodModified && <Text color="yellow">*</Text>}
              <Text bold color={isPathModified ? 'yellow' : undefined}> {displayPath}</Text>
              {isPathModified && <Text color="yellow">*</Text>}
            </>
          )}
        </Box>

        {!isTryItMode && isActive && (() => {
          const hasSaved = Object.values(parameterValues).some(v => v) || customParams.length > 0 || !!body;
          return (
            <Box flexDirection="column" flexShrink={0}>
              <Box justifyContent="flex-end">
                <Text dimColor>[ </Text>
                <Text color="cyan">Try it out (t)</Text>
                <Text dimColor> ][ </Text>
                <Text color="green" bold>Execute (e)</Text>
                {hasSaved && <Text color="yellow"> *saved params</Text>}
                <Text dimColor> ]</Text>
              </Box>
            </Box>
          );
        })()}

        {isTryItMode && isActive && (
          <Box marginTop={0} justifyContent="flex-end" flexShrink={0}>
            {hasAnyOverride && (
              <>
                <Text dimColor>[ </Text>
                <Text color="yellow">Reset (r)</Text>
                <Text dimColor> ] </Text>
              </>
            )}
            <Text dimColor>[ </Text>
            <Text color="green" bold>Execute (e)</Text>
            <Text dimColor> ] [ Cancel (Esc) ] </Text>
          </Box>
        )}

        <Box flexDirection="column" paddingX={1} width="100%" flexShrink={0}>
          {operation.summary && (
            <Box flexShrink={0}>
              <Text bold>{operation.summary}</Text>
            </Box>
          )}

          {operation.description && (
            <Box flexShrink={0}>
              <Text dimColor>{operation.description}</Text>
            </Box>
          )}

          {operation.deprecated && (
            <Box flexShrink={0}>
              <Text color="yellow" bold>DEPRECATED</Text>
            </Box>
          )}

          {(operation.parameters && operation.parameters.length > 0) || isTryItMode || customParams.length > 0 ? (
            <Box flexShrink={0} width="100%" marginTop={1} flexDirection="column">
              {(isTryItMode || headerCustomParams.length > 0) && (
                <HeadersSection
                  headers={headerCustomParams}
                  onChange={handleHeadersChange}
                  isActive={isActive && isTryItMode && headersFocused}
                  onTabOut={() => { setHeadersFocused(false); }}
                  onTabBack={() => setHeadersFocused(false)}
                  onInsertModeChange={setHeadersInsertMode}
                />
              )}
              <Box marginTop={1} flexDirection="column" width="100%">
              <ParametersSection
                parameters={operation.parameters || []}
                isTryItMode={isTryItMode}
                values={parameterValues}
                onChange={isTryItMode ? onParameterChange : undefined}
                isActive={isTryItMode && isActive && !bodyTabFocused && !editingBody && !headersFocused}
                onFocusChange={isTryItMode ? handleParamFocusChange : undefined}
                onTabOut={isTryItMode ? () => { setBodyTabFocused(true); } : undefined}
                onTabBack={isTryItMode ? () => { setHeadersFocused(true); } : undefined}
                resetKey={paramResetKey}
                customParams={nonHeaderCustomParams}
                onCustomParamsChange={(updated) => onCustomParamsChange?.([...updated, ...headerCustomParams])}
                disabledParams={disabledParams}
                onDisabledParamsChange={onDisabledParamsChange}
                onInsertModeChange={setParamInsertMode}
              />
              </Box>
            </Box>
          ) : null}

          {(operation.requestBody || ['post', 'put', 'patch'].includes(method.toLowerCase())) && (() => {
            const schema = Object.values(operation.requestBody?.content || {})[0]?.schema;
            const schemaStr = schema ? formatSchema(schema, 0, specComponents) : null;
            const schemaLines = schemaStr ? schemaStr.split('\n') : [];
            const placeholder = schema ? JSON.stringify(scaffoldPlaceholder(schema, specComponents), null, 2) : null;
            const placeholderLines = placeholder ? placeholder.split('\n') : [];

            return (
              <Box flexDirection="column" flexShrink={0} marginTop={1}>
                <Box>
                  <Text bold>BODY </Text>
                  <Text dimColor>{Object.keys(operation.requestBody?.content || {}).join(', ')}</Text>
                  {operation.requestBody?.required && (
                    <Text color="red"> *</Text>
                  )}
                </Box>

                {/* Browse mode: show overridden body if set, otherwise show schema */}
                {!isTryItMode && (
                  <Box paddingLeft={1} flexDirection="column">
                    {body ? (
                      body.split('\n').map((line, i) => (
                        <Text key={i} color="green">{line}</Text>
                      ))
                    ) : schemaLines.length > 0 ? (
                      schemaLines.map((line, i) => (
                        <Text key={i} dimColor>{line}</Text>
                      ))
                    ) : null}
                  </Box>
                )}

                {/* Try-it mode: schema inside textarea as placeholder when body empty */}
                {isTryItMode && (
                  <Box borderStyle="round" borderColor={editingBody ? 'green' : bodyTabFocused ? 'cyan' : 'gray'} paddingX={1} flexDirection="column">
                    {!editingBody && !body ? (
                      placeholderLines.length > 0 ? (
                        <>
                          {placeholderLines.map((line, i) => (
                            <Text key={i} dimColor>{line}</Text>
                          ))}
                          <Text dimColor>{bodyTabFocused ? 'i: edit | k: back' : 'j: focus'}</Text>
                        </>
                      ) : (
                        <Text dimColor>{bodyTabFocused ? 'i: edit | k: back to params' : 'j to focus, i to edit'}</Text>
                      )
                    ) : (
                      <>
                        <TextArea
                          value={body}
                          onChange={onBodyChange}
                          onSubmit={() => setEditingBodyWithSignal(false)}
                          focus={editingBody}
                          viewportLines={10}
                          onTab={(shift) => { if (!shift) onBodyChange(body + '  '); }}
                          onFirstLineUp={() => setEditingBodyWithSignal(false)}
                        />
                        {editingBody && (
                          <Text dimColor>Enter: done | Shift+Enter: newline | Esc: cancel</Text>
                        )}
                      </>
                    )}
                  </Box>
                )}
              </Box>
            );
          })()}

          <Box flexShrink={0} width="100%" marginTop={1}>
            <ResponsesSection responses={operation.responses} specComponents={specComponents} isActive={isActive && !isTryItMode} />
          </Box>

          {response && (
            <Box marginTop={1} flexShrink={0}>
              {renderResponse()}
            </Box>
          )}
        </Box>
      </Box>
    );
  };

  const renderSavedRequestDetails = (item: FlatListItem) => {
    const { method, path, name, queryParams, headers, body: savedBody, bodyType } = item.savedRequest!;
    const isTryItMode = mode === 'tryItOut';

    // Convert saved KeyValuePairs to CustomParameter[] for use with the same components
    const queryCustomParams: CustomParameter[] = (queryParams || []).map(p => ({
      id: p.id, name: p.key, value: p.value, in: 'query' as const, enabled: p.enabled,
    }));
    const headerCustomParams: CustomParameter[] = (headers || []).map(h => ({
      id: h.id, name: h.key, value: h.value, in: 'header' as const, enabled: h.enabled,
    }));

    return (
      <Box flexDirection="column">
        {/* Header */}
        <Box paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" width="100%" flexShrink={0}>
          <Text color="yellow">* </Text>
          <MethodBadge method={method} />
          <Text bold> {path}</Text>
          <Box marginLeft={2}>
            <Text dimColor>SAVED</Text>
          </Box>
        </Box>

        {/* Actions */}
        {isActive && (
          <Box justifyContent="flex-end" flexShrink={0}>
            <Text dimColor>[ Edit (E) ] [ Delete (D) ] [ </Text>
            <Text color="green" bold>Execute (e)</Text>
            <Text dimColor> ] </Text>
            <Text color="cyan">[ Try it out (t) ]</Text>
          </Box>
        )}

        {/* Content — same layout as spec endpoint */}
        <Box flexDirection="column" paddingX={1} width="100%">
          <Box flexShrink={0}>
            <Text bold>{name}</Text>
          </Box>

          {(headerCustomParams.length > 0 || queryCustomParams.length > 0) && (
            <Box flexShrink={0} width="100%" marginTop={1} flexDirection="column">
              {headerCustomParams.length > 0 && (
                <HeadersSection
                  headers={headerCustomParams}
                  onChange={() => {}}
                  isActive={false}
                  onInsertModeChange={() => {}}
                />
              )}
              {queryCustomParams.length > 0 && (
                <Box marginTop={headerCustomParams.length > 0 ? 1 : 0} flexDirection="column" width="100%">
                  <ParametersSection
                    parameters={[]}
                    isTryItMode={false}
                    values={{}}
                    customParams={queryCustomParams}
                    onCustomParamsChange={() => {}}
                    disabledParams={[]}
                    onDisabledParamsChange={() => {}}
                    onInsertModeChange={() => {}}
                  />
                </Box>
              )}
            </Box>
          )}

          {/* Body (read-only display) */}
          {savedBody && (
            <Box flexDirection="column" flexShrink={0} marginTop={1}>
              <Box>
                <Text bold>BODY </Text>
                <Text dimColor>{bodyType === 'json' ? 'application/json' : 'text/plain'}</Text>
              </Box>
              <Box paddingLeft={1} flexDirection="column">
                {savedBody.split('\n').map((line, i) => (
                  <Text key={i} color="green">{line}</Text>
                ))}
              </Box>
            </Box>
          )}

          {/* Response */}
          {response && (
            <Box marginTop={1} flexShrink={0} width="100%">
              {renderResponse()}
            </Box>
          )}
        </Box>
      </Box>
    );
  };

  const renderResponse = () => {
    if (!response) return <Box />;

    const statusColor = response.status < 300 ? 'green' : response.status < 400 ? 'yellow' : 'red';

    return (
      <Box flexDirection="column" borderStyle="single" borderColor="gray" width="100%">
        {/* Status line + tab toggle */}
        <Box paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" justifyContent="space-between">
          <Box>
            <Text bold>{responseTab === 'request' ? 'REQUEST  ' : 'RESPONSE '}</Text>
            <Text color={statusColor} bold>{response.status} {response.statusText}</Text>
            <Text dimColor> {response.time}ms</Text>
            {yankMessage && <Text color="green" bold>  [yanked]</Text>}
          </Box>
          <Box>
            {isActive && responseTab === 'response' && (() => {
              const total = response.body.split('\n').length;
              if (total > RESPONSE_VIEWPORT) {
                return <Text dimColor>[{responseCursor + 1}/{total}]  </Text>;
              }
              return null;
            })()}
            <Text color={responseTab === 'request' ? 'cyan' : undefined} dimColor={responseTab !== 'request'}>[ Request  ]</Text>
            <Text color={responseTab === 'response' ? 'cyan' : undefined} dimColor={responseTab !== 'response'}>[ Response ]</Text>
            {isActive && <Text dimColor> \:toggle</Text>}
          </Box>
        </Box>

        {/* Request tab */}
        {responseTab === 'request' ? (
          <Box flexDirection="column" paddingX={1}>
            <Text dimColor bold>{response.requestMethod} {response.requestUrl}</Text>
            {response.requestHeaders && Object.entries(response.requestHeaders).map(([k, v]) => (
              <Box key={k}>
                <Text dimColor>{k}: </Text>
                <Text>{v}</Text>
              </Box>
            ))}
            {response.requestBody && (
              <Box flexDirection="column" marginTop={1}>
                <Text dimColor bold>BODY</Text>
                <Text>{response.requestBody}</Text>
              </Box>
            )}
          </Box>
        ) : (
          <>
            {Object.keys(response.headers).length > 0 && (
              <Box flexDirection="column" paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
                {Object.entries(response.headers).map(([k, v]) => (
                  <Box key={k}>
                    <Text dimColor>{k}: </Text>
                    <Text>{v}</Text>
                  </Box>
                ))}
              </Box>
            )}
            <Box flexDirection="column" paddingX={1}>
              {(() => {
                const allLines = response.body.split('\n');
                const visibleLines = allLines.slice(responseScroll, responseScroll + RESPONSE_VIEWPORT);
                const hasMore = allLines.length > RESPONSE_VIEWPORT;
                const selStart = Math.min(visualAnchor, responseCursor);
                const selEnd = Math.max(visualAnchor, responseCursor);
                return (
                  <>
                    <Box flexDirection="column" width="100%">
                      {visibleLines.map((lineText, i) => {
                        const absLine = responseScroll + i;
                        const isCursor = absLine === responseCursor;
                        const isSelected = visualMode && absLine >= selStart && absLine <= selEnd;
                        return (
                          <Box key={absLine}>
                            <Text
                              inverse={isSelected}
                              color={!isSelected && isCursor ? 'cyan' : undefined}
                            >
                              {lineText || ' '}
                            </Text>
                          </Box>
                        );
                      })}
                    </Box>
                    {isActive && (
                      <Box>
                        {visualMode
                          ? <Text color="cyan">-- VISUAL --  y: yank selection  Esc: cancel</Text>
                          : hasMore
                          ? <Text dimColor>J/K: move  g/G: top/bottom  v: visual  y: yank all</Text>
                          : <Text dimColor>v: visual  y: yank</Text>
                        }
                      </Box>
                    )}
                  </>
                );
              })()}
            </Box>
            {curl && (
              <Box paddingX={1}>
                <Text dimColor>curl: {curl}</Text>
              </Box>
            )}
          </>
        )}
      </Box>
    );
  };

  return (
    <Box
      flexDirection="column"
      borderStyle={isActive ? 'bold' : 'single'}
      borderColor={isActive ? 'cyan' : 'gray'}
      flexGrow={1}
      height={height}
    >
      {isLoading ? (
        <Box paddingX={1}>
          <Text color="cyan">Executing request...</Text>
        </Box>
      ) : (
        <Box flexDirection="column" overflowY="hidden" height={height - 2}>
          <Box flexDirection="column" marginTop={-scrollOffset}>
            {renderContent()}
          </Box>
        </Box>
      )}
    </Box>
  );
}

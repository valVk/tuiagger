import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { TextArea } from 'react-ink-textarea';
import type { HttpMethodType } from '../types/index.js';
import { MethodBadge } from './MethodBadge.js';
import { ParametersSection } from './ParametersSection.js';
import { HeadersSection } from './HeadersSection.js';
import { ResponsesSection } from './ResponsesSection.js';
import { ResponseViewer } from './ResponseViewer.js';
import type { FlatListItem, RightPanelMode } from '../hooks/usePanelNavigation.js';
import type { ResponseState, CustomParameter } from '../types/index.js';
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
  customParams?: CustomParameter[];
  onCustomParamsChange?: (params: CustomParameter[]) => void;
  disabledParams?: string[];
  onDisabledParamsChange?: (disabled: string[]) => void;
  overridePath?: string;
  overrideMethod?: string;
  onOverridePathChange?: (path: string) => void;
  onOverrideMethodChange?: (method: string) => void;
  onResetOverrides?: () => void;
  showResetConfirm?: boolean;
  onResetConfirmResponse?: (confirmed: boolean) => void;
  onEditingChange?: (editing: boolean) => void;
  specComponents?: Record<string, unknown>;
  onNormalModeChange?: (isNormal: boolean) => void;
}

const LINES_PER_PARAM = 2;
const HEADER_LINES = 10;

const HTTP_METHODS: HttpMethodType[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

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
  onNormalModeChange,
}: RightPanelProps) {
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

  const renderContent = () => {
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

    const nonHeaderCustomParams = customParams.filter(p => p.in !== 'header');
    const headerCustomParams = customParams.filter(p => p.in === 'header');

    const handleHeadersChange = (updated: CustomParameter[]) => {
      onCustomParamsChange?.([...nonHeaderCustomParams, ...updated]);
    };

    const displayMethod = overrideMethod || method;
    const displayPath = overridePath || path;
    const isMethodModified = overrideMethod && overrideMethod !== method;
    const isPathModified = overridePath && overridePath !== path;
    const hasAnyOverride = isMethodModified || isPathModified;

    return (
      <Box flexDirection="column">
        {showResetConfirm && (
          <Box paddingX={1} paddingY={1} borderStyle="double" borderColor="yellow" width="100%">
            <Text color="yellow" bold>Reset all overrides for this endpoint? (y/n)</Text>
          </Box>
        )}

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
                  <TextInput value={displayPath} onChange={(v) => onOverridePathChange?.(v)} focus={true} />
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
            <Box flexShrink={0}><Text bold>{operation.summary}</Text></Box>
          )}
          {operation.description && (
            <Box flexShrink={0}><Text dimColor>{operation.description}</Text></Box>
          )}
          {operation.deprecated && (
            <Box flexShrink={0}><Text color="yellow" bold>DEPRECATED</Text></Box>
          )}

          {((operation.parameters && operation.parameters.length > 0) || isTryItMode || customParams.length > 0) && (
            <Box flexShrink={0} width="100%" marginTop={1} flexDirection="column">
              {(isTryItMode || headerCustomParams.length > 0) && (
                <HeadersSection
                  headers={headerCustomParams}
                  onChange={handleHeadersChange}
                  isActive={isActive && isTryItMode && headersFocused}
                  onTabOut={() => setHeadersFocused(false)}
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
                  onFocusChange={isTryItMode ? scrollToParamRow : undefined}
                  onTabOut={isTryItMode ? () => setBodyTabFocused(true) : undefined}
                  onTabBack={isTryItMode ? () => setHeadersFocused(true) : undefined}
                  resetKey={paramResetKey}
                  customParams={nonHeaderCustomParams}
                  onCustomParamsChange={(updated) => onCustomParamsChange?.([...updated, ...headerCustomParams])}
                  disabledParams={disabledParams}
                  onDisabledParamsChange={onDisabledParamsChange}
                  onInsertModeChange={setParamInsertMode}
                />
              </Box>
            </Box>
          )}

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
                  {operation.requestBody?.required && <Text color="red"> *</Text>}
                </Box>

                {!isTryItMode && (
                  <Box paddingLeft={1} flexDirection="column">
                    {body ? (
                      body.split('\n').map((line, i) => <Text key={i} color="green">{line}</Text>)
                    ) : schemaLines.length > 0 ? (
                      schemaLines.map((line, i) => <Text key={i} dimColor>{line}</Text>)
                    ) : null}
                  </Box>
                )}

                {isTryItMode && (
                  <Box borderStyle="round" borderColor={editingBody ? 'green' : bodyTabFocused ? 'cyan' : 'gray'} paddingX={1} flexDirection="column">
                    {!editingBody && !body ? (
                      placeholderLines.length > 0 ? (
                        <>
                          {placeholderLines.map((line, i) => <Text key={i} dimColor>{line}</Text>)}
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
              <ResponseViewer
                response={response}
                curl={curl}
                isActive={isActive && !isTryItMode}
                onScrollReset={() => onScrollChange?.(0)}
              />
            </Box>
          )}
        </Box>
      </Box>
    );
  };

  const renderSavedRequestDetails = (item: FlatListItem) => {
    const { method, path, name, queryParams, headers, body: savedBody, bodyType } = item.savedRequest!;
    const isTryItMode = mode === 'tryItOut';

    const queryCustomParams: CustomParameter[] = (queryParams || []).map(p => ({
      id: p.id, name: p.key, value: p.value, in: 'query' as const, enabled: p.enabled,
    }));
    const headerCustomParams: CustomParameter[] = (headers || []).map(h => ({
      id: h.id, name: h.key, value: h.value, in: 'header' as const, enabled: h.enabled,
    }));

    return (
      <Box flexDirection="column">
        <Box paddingX={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray" width="100%" flexShrink={0}>
          <Text color="yellow">* </Text>
          <MethodBadge method={method} />
          <Text bold> {path}</Text>
          <Box marginLeft={2}><Text dimColor>SAVED</Text></Box>
        </Box>

        {isActive && (
          <Box justifyContent="flex-end" flexShrink={0}>
            <Text dimColor>[ Edit (E) ] [ Delete (D) ] [ </Text>
            <Text color="green" bold>Execute (e)</Text>
            <Text dimColor> ] </Text>
            <Text color="cyan">[ Try it out (t) ]</Text>
          </Box>
        )}

        <Box flexDirection="column" paddingX={1} width="100%">
          <Box flexShrink={0}><Text bold>{name}</Text></Box>

          {(headerCustomParams.length > 0 || queryCustomParams.length > 0) && (
            <Box flexShrink={0} width="100%" marginTop={1} flexDirection="column">
              {headerCustomParams.length > 0 && (
                <HeadersSection headers={headerCustomParams} onChange={() => {}} isActive={false} onInsertModeChange={() => {}} />
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

          {savedBody && (
            <Box flexDirection="column" flexShrink={0} marginTop={1}>
              <Box>
                <Text bold>BODY </Text>
                <Text dimColor>{bodyType === 'json' ? 'application/json' : 'text/plain'}</Text>
              </Box>
              <Box paddingLeft={1} flexDirection="column">
                {savedBody.split('\n').map((line, i) => <Text key={i} color="green">{line}</Text>)}
              </Box>
            </Box>
          )}

          {response && (
            <Box marginTop={1} flexShrink={0} width="100%">
              <ResponseViewer
                response={response}
                curl={curl}
                isActive={isActive}
                onScrollReset={() => onScrollChange?.(0)}
              />
            </Box>
          )}
        </Box>
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

import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { TextArea } from 'react-ink-textarea';
import type { HttpMethodType, ResponseState, CustomParameter, SavedRequest } from '../types/index.js';
import { MethodBadge } from './MethodBadge.js';
import { ParametersSection } from './ParametersSection.js';
import { HeadersSection } from './HeadersSection.js';
import { ResponseViewer } from './ResponseViewer.js';

const HTTP_METHODS: HttpMethodType[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

interface ManualRequestPanelProps {
  path: string;
  method: string;
  customParams: CustomParameter[];
  body: string;
  editingRequest?: SavedRequest;
  isActive: boolean;
  response?: ResponseState | null;
  curl?: string;
  isLoading?: boolean;
  height?: number;
  scrollOffset?: number;
  onPathChange: (path: string) => void;
  onMethodChange: (method: string) => void;
  onCustomParamsChange: (params: CustomParameter[]) => void;
  onBodyChange: (body: string) => void;
  onNormalModeChange?: (isNormal: boolean) => void;
  onScrollReset?: () => void;
}

export function ManualRequestPanel({
  path,
  method,
  customParams,
  body,
  editingRequest,
  isActive,
  response,
  curl,
  isLoading,
  height = 20,
  scrollOffset = 0,
  onPathChange,
  onMethodChange,
  onCustomParamsChange,
  onBodyChange,
  onNormalModeChange,
  onScrollReset,
}: ManualRequestPanelProps) {
  const [editingPath, setEditingPath] = useState(false);
  const [bodyTabFocused, setBodyTabFocused] = useState(false);
  const [editingBody, setEditingBody] = useState(false);
  const [paramInsertMode, setParamInsertMode] = useState(false);
  const [headersFocused, setHeadersFocused] = useState(false);
  const [headersInsertMode, setHeadersInsertMode] = useState(false);

  useEffect(() => {
    onNormalModeChange?.(!editingPath && !editingBody && !paramInsertMode && !headersInsertMode);
  }, [editingPath, editingBody, paramInsertMode, headersInsertMode]);

  const nonHeaderParams = customParams.filter(p => p.in !== 'header');
  const headerParams = customParams.filter(p => p.in === 'header');

  const handleHeadersChange = (updated: CustomParameter[]) => {
    onCustomParamsChange([...nonHeaderParams, ...updated]);
  };

  const displayMethod = method || 'GET';
  const displayPath = path ?? '';

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
        const currentIndex = HTTP_METHODS.indexOf(displayMethod.toUpperCase() as HttpMethodType);
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

  if (isLoading) {
    return (
      <Box
        flexDirection="column"
        borderStyle="bold"
        borderColor="cyan"
        flexGrow={1}
        height={height}
      >
        <Box paddingX={1}>
          <Text color="cyan">Executing request...</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box
      flexDirection="column"
      borderStyle={isActive ? 'bold' : 'single'}
      borderColor={isActive ? 'cyan' : 'gray'}
      flexGrow={1}
      height={height}
    >
      <Box flexDirection="column" overflowY="hidden" height={height - 2}>
        <Box flexDirection="column" marginTop={-scrollOffset}>
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
              <Box flexGrow={1}>
                <TextInput
                  value={displayPath}
                  onChange={onPathChange}
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
                headers={headerParams}
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
                onFocusChange={() => {}}
                onTabOut={['POST', 'PUT', 'PATCH', 'DELETE'].includes(displayMethod.toUpperCase()) ? () => setBodyTabFocused(true) : undefined}
                onTabBack={() => setHeadersFocused(true)}
                customParams={nonHeaderParams}
                onCustomParamsChange={(updated) => onCustomParamsChange([...updated, ...headerParams])}
                disabledParams={[]}
                onDisabledParamsChange={() => {}}
                onInsertModeChange={setParamInsertMode}
              />
            </Box>

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
                        onSubmit={() => setEditingBody(false)}
                        focus={editingBody}
                        viewportLines={10}
                        onTab={(shift) => { if (!shift) onBodyChange(body + '  '); }}
                        onFirstLineUp={() => setEditingBody(false)}
                      />
                      {editingBody && (
                        <Text dimColor>Enter: done  Shift+Enter: newline  Esc: cancel</Text>
                      )}
                    </>
                  )}
                </Box>
              </Box>
            )}

            {response && (
              <Box marginTop={1} flexShrink={0}>
                <ResponseViewer
                  response={response}
                  curl={curl}
                  isActive={isActive}
                  onScrollReset={onScrollReset}
                />
              </Box>
            )}
          </Box>
        </Box>
      </Box>
    </Box>
  );
}

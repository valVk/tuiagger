import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { spawn } from 'child_process';
import type { ResponseState } from '../types/index.js';

const RESPONSE_VIEWPORT = 15;

interface ResponseViewerProps {
  response: ResponseState;
  curl?: string;
  isActive: boolean;
  onScrollReset?: () => void;
}

export function ResponseViewer({ response, curl, isActive, onScrollReset }: ResponseViewerProps) {
  const [responseTab, setResponseTab] = useState<'response' | 'request'>('response');
  const [responseScroll, setResponseScroll] = useState(0);
  const [responseCursor, setResponseCursor] = useState(0);
  const [visualMode, setVisualMode] = useState(false);
  const [visualAnchor, setVisualAnchor] = useState(0);
  const [yankMessage, setYankMessage] = useState(false);

  useEffect(() => {
    setResponseScroll(0);
    setResponseCursor(0);
    setVisualMode(false);
    setVisualAnchor(0);
    setYankMessage(false);
  }, [response]);

  const statusColor = response.status < 300 ? 'green' : response.status < 400 ? 'yellow' : 'red';

  useInput(
    (input, key) => {
      if (input === '\\') {
        setResponseTab(prev => prev === 'response' ? 'request' : 'response');
        onScrollReset?.();
        return;
      }

      if (responseTab !== 'response') return;

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

      if (key.escape) { setVisualMode(false); return; }
      if (input === 'v') {
        setVisualMode(v => { if (!v) setVisualAnchor(responseCursor); return !v; });
        return;
      }
      if (input === 'J') { moveCursor(responseCursor + 1); return; }
      if (input === 'K') { moveCursor(responseCursor - 1); return; }
      if (input === 'G') { moveCursor(maxLine); return; }
      if (input === 'g') { moveCursor(0); return; }

      if (input === 'y') {
        const bodyLines2 = response.body.split('\n');
        let textToCopy: string;
        if (visualMode) {
          const selStart = Math.min(visualAnchor, responseCursor);
          const selEnd = Math.max(visualAnchor, responseCursor);
          textToCopy = bodyLines2.slice(selStart, selEnd + 1).join('\n');
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
    { isActive }
  );

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
                          <Text inverse={isSelected} color={!isSelected && isCursor ? 'cyan' : undefined}>
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
}

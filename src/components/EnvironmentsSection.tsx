import React from 'react';
import { Box, Text } from 'ink';
import TextInput from 'ink-text-input';
import type { EnvironmentsStore } from '../types/index.js';

type EnvView = 'list' | 'edit';
type VarField = 'key' | 'value';

const NAME_W = 24;
const VALUE_W = 30;

export interface EnvironmentsSectionProps {
  environments: EnvironmentsStore;
  activeEnvIdx: number;
  envView: EnvView;
  envCursor: number;
  varCursor: number;
  insertingVar: boolean;
  editingVarKey: string | null;
  varField: VarField;
  varKeyInput: string;
  varValueInput: string;
  addingEnv: boolean;
  newEnvName: string;
  isActive: boolean;
  onNewEnvNameChange: (v: string) => void;
  onVarKeyInputChange: (v: string) => void;
  onVarValueInputChange: (v: string) => void;
}

export function EnvironmentsSection({
  environments,
  activeEnvIdx,
  envView,
  envCursor,
  varCursor,
  insertingVar,
  editingVarKey,
  varField,
  varKeyInput,
  varValueInput,
  addingEnv,
  newEnvName,
  isActive,
  onNewEnvNameChange,
  onVarKeyInputChange,
  onVarValueInputChange,
}: EnvironmentsSectionProps) {
  const envList = environments.environments;

  return (
    <>
      <Box paddingX={1} flexShrink={0} marginTop={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
        <Text bold color={isActive ? 'cyan' : undefined}>ENVIRONMENTS  </Text>
        {isActive && envView === 'list' && (
          <Text dimColor>Tab: switch  j/k: move  Enter: activate  e: edit  n: new  x: del</Text>
        )}
        {isActive && envView === 'edit' && (
          <Text dimColor>j/k: move  i: edit  x: del  Esc: back</Text>
        )}
      </Box>

      {isActive && envView === 'edit' ? (
        (() => {
          const env = envList[envCursor];
          if (!env) return null;
          const varEntries = Object.entries(env.variables);
          return (
            <Box flexDirection="column" paddingX={1}>
              <Box marginBottom={0}>
                <Text bold>Variables for </Text>
                <Text bold color="cyan">{env.name}</Text>
              </Box>
              <Box>
                <Box width={3} flexShrink={0} />
                <Box width={NAME_W} flexShrink={0}><Text dimColor bold>NAME</Text></Box>
                <Box width={VALUE_W} flexShrink={0}><Text dimColor bold>VALUE</Text></Box>
              </Box>
              {varEntries.map(([k, v], i) => {
                const isCur = i === varCursor;
                const isEditing = insertingVar && editingVarKey === k;
                return (
                  <Box key={k}>
                    <Box width={3} flexShrink={0}><Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text></Box>
                    <Box width={NAME_W} flexShrink={0}>
                      {isEditing && varField === 'key'
                        ? <TextInput value={varKeyInput} onChange={onVarKeyInputChange} focus={true} />
                        : <Text color={isCur ? 'cyan' : undefined}>{k}</Text>
                      }
                    </Box>
                    <Box width={VALUE_W} flexShrink={0}>
                      {isEditing && varField === 'value'
                        ? <TextInput value={varValueInput} onChange={onVarValueInputChange} focus={true} />
                        : <Text color="green">{v || '-'}</Text>
                      }
                    </Box>
                  </Box>
                );
              })}
              {(() => {
                const isCur = varCursor === varEntries.length;
                const isAdding = insertingVar && editingVarKey === null;
                return (
                  <Box>
                    <Box width={3} flexShrink={0}><Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text></Box>
                    {isAdding ? (
                      <>
                        <Box width={NAME_W} flexShrink={0}>
                          {varField === 'key'
                            ? <TextInput value={varKeyInput} onChange={onVarKeyInputChange} focus={true} placeholder="var-name" />
                            : <Text color="cyan">{varKeyInput || '-'}</Text>
                          }
                        </Box>
                        <Box width={VALUE_W} flexShrink={0}>
                          {varField === 'value'
                            ? <TextInput value={varValueInput} onChange={onVarValueInputChange} focus={true} placeholder="value" />
                            : <Text dimColor>-</Text>
                          }
                        </Box>
                      </>
                    ) : (
                      <Text color={isCur ? 'cyan' : undefined} dimColor={!isCur}>
                        {isCur ? '[ i: add variable ]' : '[ + ]'}
                      </Text>
                    )}
                  </Box>
                );
              })()}
            </Box>
          );
        })()
      ) : (
        <Box flexDirection="column" paddingX={1}>
          {envList.map((env, i) => {
            const isCur = isActive && i === envCursor;
            const isEnvActive = i === activeEnvIdx;
            return (
              <Box key={i}>
                <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
                <Text color={isCur ? 'cyan' : undefined}>{env.name}</Text>
                {isEnvActive && <Text color="green">  active</Text>}
              </Box>
            );
          })}
          {isActive && (
            addingEnv ? (
              <Box>
                <Text color="cyan">{'> '}</Text>
                <TextInput value={newEnvName} onChange={onNewEnvNameChange} focus={true} placeholder="env name" />
              </Box>
            ) : (
              <Box>
                <Text dimColor>  </Text>
                <Text dimColor>[ n: new environment ]</Text>
              </Box>
            )
          )}
          {envList.length === 0 && !addingEnv && (
            <Text dimColor>  no environments — press n to create one</Text>
          )}
        </Box>
      )}
    </>
  );
}

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import type { OpenAPISpec, EnvironmentsStore } from '../types/index.js';

interface InfoPopupProps {
  spec: OpenAPISpec;
  selectedServer: number;
  onServerChange: (index: number) => void;
  onClose: () => void;
  collectionName?: string;
  credentials: Record<string, string>;
  onSetCredential: (name: string, val: string) => void;
  environments: EnvironmentsStore;
  onAddEnvironment: (name: string) => void;
  onDeleteEnvironment: (index: number) => void;
  onSetActive: (index: number) => void;
  onSetVariable: (envIndex: number, key: string, value: string) => void;
  onDeleteVariable: (envIndex: number, key: string) => void;
}

type Section = 'servers' | 'auth' | 'environments';
type EnvView = 'list' | 'edit';
type VarField = 'key' | 'value';

const NAME_W = 24;
const VALUE_W = 30;

export function InfoPopup({
  spec,
  selectedServer,
  onServerChange,
  onClose,
  collectionName,
  credentials,
  onSetCredential,
  environments,
  onAddEnvironment,
  onDeleteEnvironment,
  onSetActive,
  onSetVariable,
  onDeleteVariable,
}: InfoPopupProps) {
  const servers = spec.servers || [{ url: 'http://localhost', description: 'Default' }];
  const securitySchemes = spec.components?.securitySchemes
    ? Object.entries(spec.components.securitySchemes)
    : [];
  const envList = environments.environments;

  const [section, setSection] = useState<Section>('servers');
  const [cursor, setCursor] = useState(selectedServer);
  const [authCursor, setAuthCursor] = useState(0);
  const [editingScheme, setEditingScheme] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');

  // Environments state
  const [envView, setEnvView] = useState<EnvView>('list');
  const [envCursor, setEnvCursor] = useState(0);
  const [addingEnv, setAddingEnv] = useState(false);
  const [newEnvName, setNewEnvName] = useState('');
  const [varCursor, setVarCursor] = useState(0);
  const [insertingVar, setInsertingVar] = useState(false);
  const [varField, setVarField] = useState<VarField>('key');
  const [editingVarKey, setEditingVarKey] = useState<string | null>(null); // null = adding new
  const [varKeyInput, setVarKeyInput] = useState('');
  const [varValueInput, setVarValueInput] = useState('');

  const isInsertMode = !!(editingScheme || addingEnv || insertingVar);

  const cycleSections = () => {
    const order: Section[] = ['servers', 'auth', 'environments'];
    const filtered = order.filter(s => s !== 'auth' || securitySchemes.length > 0);
    const idx = filtered.indexOf(section);
    setSection(filtered[(idx + 1) % filtered.length]);
  };

  useInput((input, key) => {
    // === AUTH edit mode ===
    if (editingScheme) {
      if (key.escape || key.return) {
        onSetCredential(editingScheme, editValue);
        setEditingScheme(null);
      }
      return;
    }

    // === Adding new env name ===
    if (addingEnv) {
      if (key.return && newEnvName.trim()) {
        onAddEnvironment(newEnvName.trim());
        setEnvCursor(envList.length); // will be new item
        setAddingEnv(false);
        setNewEnvName('');
      } else if (key.escape) {
        setAddingEnv(false);
        setNewEnvName('');
      }
      return;
    }

    // === Inserting/editing variable ===
    if (insertingVar) {
      if (key.tab) {
        setVarField(f => f === 'key' ? 'value' : 'key');
        return;
      }
      if (key.escape) {
        setInsertingVar(false);
        setEditingVarKey(null);
        setVarKeyInput('');
        setVarValueInput('');
        return;
      }
      if (key.return) {
        if (varKeyInput.trim()) {
          onSetVariable(envCursor, varKeyInput.trim(), varValueInput);
        }
        setInsertingVar(false);
        setEditingVarKey(null);
        setVarKeyInput('');
        setVarValueInput('');
      }
      return;
    }

    // === Global close / back ===
    const isEnvEdit = section === 'environments' && envView === 'edit';
    if (key.escape || (input === 'i' && !isEnvEdit)) {
      if (envView === 'edit') {
        setEnvView('list');
        setVarCursor(0);
        return;
      }
      onClose();
      return;
    }

    // === Tab cycles sections ===
    if (key.tab) {
      cycleSections();
      return;
    }

    // === SERVERS section ===
    if (section === 'servers') {
      if (input === 'j' || key.downArrow) { setCursor(p => Math.min(p + 1, servers.length - 1)); return; }
      if (input === 'k' || key.upArrow)   { setCursor(p => Math.max(p - 1, 0)); return; }
      if (key.return) { onServerChange(cursor); onClose(); return; }
    }

    // === AUTH section ===
    if (section === 'auth') {
      if (input === 'j' || key.downArrow) { setAuthCursor(p => Math.min(p + 1, securitySchemes.length - 1)); return; }
      if (input === 'k' || key.upArrow)   { setAuthCursor(p => Math.max(p - 1, 0)); return; }
      if (key.return && securitySchemes.length > 0) {
        const [name] = securitySchemes[authCursor];
        setEditValue(credentials[name] || '');
        setEditingScheme(name);
        return;
      }
    }

    // === ENVIRONMENTS section ===
    if (section === 'environments') {
      if (envView === 'list') {
        if (input === 'j' || key.downArrow) { setEnvCursor(p => Math.min(p + 1, envList.length - 1)); return; }
        if (input === 'k' || key.upArrow)   { setEnvCursor(p => Math.max(p - 1, 0)); return; }
        if (key.return && envList.length > 0) { onSetActive(envCursor); return; }
        if (input === 'e' && envList.length > 0) {
          setEnvView('edit');
          setVarCursor(0);
          return;
        }
        if (input === 'n') {
          setAddingEnv(true);
          setNewEnvName('');
          return;
        }
        if (input === 'x' && envList.length > 0) {
          onDeleteEnvironment(envCursor);
          setEnvCursor(c => Math.max(0, c - 1));
          return;
        }
      }

      if (envView === 'edit') {
        const env = envList[envCursor];
        if (!env) return;
        const varEntries = Object.entries(env.variables);
        const totalVarRows = varEntries.length + 1; // +1 for addNew

        if (input === 'j' || key.downArrow) { setVarCursor(p => Math.min(p + 1, totalVarRows - 1)); return; }
        if (input === 'k' || key.upArrow)   { setVarCursor(p => Math.max(p - 1, 0)); return; }

        if (input === 'i') {
          if (varCursor < varEntries.length) {
            const [k, v] = varEntries[varCursor];
            setEditingVarKey(k);
            setVarKeyInput(k);
            setVarValueInput(v);
          } else {
            setEditingVarKey(null);
            setVarKeyInput('');
            setVarValueInput('');
          }
          setVarField('key');
          setInsertingVar(true);
          return;
        }

        if (input === 'x' && varCursor < varEntries.length) {
          onDeleteVariable(envCursor, varEntries[varCursor][0]);
          setVarCursor(c => Math.max(0, c - 1));
          return;
        }
      }
    }
  });

  const activeEnvIdx = environments.activeIndex;

  return (
    <Box flexDirection="column" borderStyle="double" borderColor="cyan">

      {/* Title */}
      <Box paddingX={1} flexShrink={0}>
        {collectionName && <Text color="yellow" bold>[{collectionName}]  </Text>}
        <Text bold wrap="truncate">{spec.info.title}</Text>
        <Text dimColor>  v{spec.info.version}  </Text>
        <Text dimColor>OpenAPI {spec.openapi}</Text>
      </Box>

      {spec.info.description && (
        <Box paddingX={1} flexShrink={0}>
          <Text dimColor wrap="truncate">{spec.info.description}</Text>
        </Box>
      )}

      {/* SERVERS */}
      <Box paddingX={1} flexShrink={0} marginTop={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
        <Text bold color={section === 'servers' ? 'cyan' : undefined}>SERVERS  </Text>
        {section === 'servers' && <Text dimColor>Tab: switch  j/k: move  Enter: select  Esc: close</Text>}
      </Box>
      {servers.map((server, i) => {
        const isActive = i === selectedServer;
        const isCur = section === 'servers' && i === cursor;
        return (
          <Box key={i} paddingX={1} flexShrink={0}>
            <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
            <Text color={isCur ? 'cyan' : undefined} bold={isActive}>{server.url}</Text>
            {server.description && <Text dimColor>  {server.description}</Text>}
            {isActive && <Text color="green">  active</Text>}
          </Box>
        );
      })}

      {/* AUTH */}
      {securitySchemes.length > 0 && (
        <>
          <Box paddingX={1} flexShrink={0} marginTop={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
            <Text bold color={section === 'auth' ? 'cyan' : undefined}>AUTH  </Text>
            {section === 'auth' && <Text dimColor>Tab: switch  j/k: move  Enter: edit  Esc: close</Text>}
          </Box>
          {securitySchemes.map(([name, scheme], i) => {
            const isCur = section === 'auth' && i === authCursor;
            const val = credentials[name] || '';
            const isEditing = editingScheme === name;
            let schemeLabel: string = scheme.type;
            if (scheme.type === 'http') schemeLabel = scheme.scheme ? `${scheme.scheme}${scheme.bearerFormat ? ` (${scheme.bearerFormat})` : ''}` : 'http';
            else if (scheme.type === 'apiKey') schemeLabel = `apiKey in ${scheme.in} as ${scheme.name}`;
            return (
              <Box key={name} paddingX={1} flexShrink={0}>
                <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
                <Text color={isCur ? 'cyan' : undefined} bold>{name}</Text>
                <Text dimColor>  {schemeLabel}  </Text>
                {isEditing
                  ? <TextInput value={editValue} onChange={setEditValue} focus={true} />
                  : val
                  ? <Text color="green">{val.length > 20 ? val.slice(0, 20) + '…' : val}</Text>
                  : <Text dimColor>not set</Text>
                }
              </Box>
            );
          })}
        </>
      )}

      {/* ENVIRONMENTS */}
      <Box paddingX={1} flexShrink={0} marginTop={1} borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderColor="gray">
        <Text bold color={section === 'environments' ? 'cyan' : undefined}>ENVIRONMENTS  </Text>
        {section === 'environments' && envView === 'list' && (
          <Text dimColor>Tab: switch  j/k: move  Enter: activate  e: edit  n: new  x: del</Text>
        )}
        {section === 'environments' && envView === 'edit' && (
          <Text dimColor>j/k: move  i: edit  x: del  Esc: back</Text>
        )}
      </Box>

      {section === 'environments' && envView === 'edit' ? (
        /* Variable editor */
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
              {/* Table header */}
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
                        ? <TextInput value={varKeyInput} onChange={setVarKeyInput} focus={true} />
                        : <Text color={isCur ? 'cyan' : undefined}>{k}</Text>
                      }
                    </Box>
                    <Box width={VALUE_W} flexShrink={0}>
                      {isEditing && varField === 'value'
                        ? <TextInput value={varValueInput} onChange={setVarValueInput} focus={true} />
                        : <Text color="green">{v || '-'}</Text>
                      }
                    </Box>
                  </Box>
                );
              })}
              {/* Add new var row */}
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
                            ? <TextInput value={varKeyInput} onChange={setVarKeyInput} focus={true} placeholder="var-name" />
                            : <Text color="cyan">{varKeyInput || '-'}</Text>
                          }
                        </Box>
                        <Box width={VALUE_W} flexShrink={0}>
                          {varField === 'value'
                            ? <TextInput value={varValueInput} onChange={setVarValueInput} focus={true} placeholder="value" />
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
        /* Environment list */
        <Box flexDirection="column" paddingX={1}>
          {envList.map((env, i) => {
            const isCur = section === 'environments' && i === envCursor;
            const isActive = i === activeEnvIdx;
            return (
              <Box key={i}>
                <Text color={isCur ? 'cyan' : 'gray'}>{isCur ? '> ' : '  '}</Text>
                <Text color={isCur ? 'cyan' : undefined}>{env.name}</Text>
                {isActive && <Text color="green">  active</Text>}
              </Box>
            );
          })}
          {section === 'environments' && (
            addingEnv ? (
              <Box>
                <Text color="cyan">{'> '}</Text>
                <TextInput value={newEnvName} onChange={setNewEnvName} focus={true} placeholder="env name" />
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

    </Box>
  );
}

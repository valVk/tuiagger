import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import type { OpenAPISpec, EnvironmentsStore, SecuritySchemeObject } from '../types/index.js';
import { useServersKeyboard } from '../hooks/useServersKeyboard.js';
import { useAuthKeyboard } from '../hooks/useAuthKeyboard.js';
import { useEnvironmentsKeyboard } from '../hooks/useEnvironmentsKeyboard.js';
import { ServersSection } from './ServersSection.js';
import { AuthSection } from './AuthSection.js';
import { EnvironmentsSection } from './EnvironmentsSection.js';

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
    ? Object.entries(spec.components.securitySchemes) as [string, SecuritySchemeObject][]
    : [];

  const [section, setSection] = useState<Section>('servers');

  const cycleSections = () => {
    const order: Section[] = ['servers', 'auth', 'environments'];
    const filtered = order.filter(s => s !== 'auth' || securitySchemes.length > 0);
    const idx = filtered.indexOf(section);
    setSection(filtered[(idx + 1) % filtered.length]);
  };

  const serversKb = useServersKeyboard({
    serversCount: servers.length,
    initialCursor: selectedServer,
    isActive: section === 'servers',
    onServerChange,
    onClose,
  });

  const authKb = useAuthKeyboard({
    securitySchemes,
    credentials,
    isActive: section === 'auth',
    onSetCredential,
  });

  const envKb = useEnvironmentsKeyboard({
    environments,
    isActive: section === 'environments',
    onSetActive,
    onAddEnvironment,
    onDeleteEnvironment,
    onSetVariable,
    onDeleteVariable,
  });

  const isInsertMode = authKb.isInsertMode || envKb.isInsertMode;

  useInput((input, key) => {
    if (isInsertMode) return;

    if (key.escape || (input === 'i' && !(section === 'environments' && envKb.envView === 'edit'))) {
      if (envKb.envView === 'edit') {
        envKb.setEnvView('list');
        return;
      }
      onClose();
      return;
    }

    if (key.tab) {
      cycleSections();
      return;
    }
  });

  return (
    <Box flexDirection="column" borderStyle="double" borderColor="cyan">
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

      <ServersSection
        servers={servers}
        selectedServer={selectedServer}
        cursor={serversKb.cursor}
        isActive={section === 'servers'}
      />

      <AuthSection
        securitySchemes={securitySchemes}
        credentials={credentials}
        authCursor={authKb.authCursor}
        editingScheme={authKb.editingScheme}
        editValue={authKb.editValue}
        isActive={section === 'auth'}
        onEditValueChange={authKb.setEditValue}
      />

      <EnvironmentsSection
        environments={environments}
        activeEnvIdx={environments.activeIndex}
        envView={envKb.envView}
        envCursor={envKb.envCursor}
        varCursor={envKb.varCursor}
        insertingVar={envKb.insertingVar}
        editingVarKey={envKb.editingVarKey}
        varField={envKb.varField}
        varKeyInput={envKb.varKeyInput}
        varValueInput={envKb.varValueInput}
        addingEnv={envKb.addingEnv}
        newEnvName={envKb.newEnvName}
        isActive={section === 'environments'}
        onNewEnvNameChange={envKb.setNewEnvName}
        onVarKeyInputChange={envKb.setVarKeyInput}
        onVarValueInputChange={envKb.setVarValueInput}
      />
    </Box>
  );
}

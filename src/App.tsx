import React, { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Box, Text, useInput, useApp } from 'ink';
import { useScreenSize } from 'fullscreen-ink';
import {
  InfoPopup,
  HelpPopup,
  Header,
  StatusBar,
  LeftPanel,
  RightPanel,
  ManualRequest,
  Spinner,
} from './components/index.js';
import {
  useOpenAPI,
  useSavedRequests,
  useRequest,
  usePanelNavigation,
  useOverrides,
  useAuth,
  useEnvironments,
} from './hooks/index.js';
import { getEndpointsByTag } from './utils/parser.js';
import { scaffoldBody } from './utils/scaffoldBody.js';
import { interpolate } from './utils/interpolate.js';
import type { KeyValuePair, SavedRequest, CustomParameter, SecuritySchemeObject } from './types/index.js';

interface AppProps {
  source: string;
  collectionName?: string;
}

type AppMode = 'browse' | 'tryit' | 'manual';

export function App({ source, collectionName }: AppProps) {
  const { exit } = useApp();
  const { height: terminalHeight } = useScreenSize();
  const { spec, loading, error, reload } = useOpenAPI(source);
  const savedRequests = useSavedRequests();
  const request = useRequest();
  const overrides = useOverrides();
  const auth = useAuth();
  const envs = useEnvironments();
  const activeEnvVarsRef = useRef(envs.activeEnv?.variables);
  useEffect(() => {
    activeEnvVarsRef.current = envs.activeEnv?.variables;
  }, [envs.activeEnv]);

  // FullScreenBox sets outer height = stdout.rows; statusbar takes 3 rows
  const contentHeight = Math.max(terminalHeight - 6, 10);

  const [mode, setMode] = useState<AppMode>('browse');
  const [selectedServer, setSelectedServer] = useState(0);
  const [parameterValues, setParameterValues] = useState<Record<string, string>>({});
  const [customParams, setCustomParams] = useState<CustomParameter[]>([]);
  const [disabledParams, setDisabledParams] = useState<string[]>([]);
  const [body, setBody] = useState('');
  const [overridePath, setOverridePath] = useState<string | undefined>();
  const [overrideMethod, setOverrideMethod] = useState<string | undefined>();
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [showInfoPopup, setShowInfoPopup] = useState(false);
  const [showHelpPopup, setShowHelpPopup] = useState(false);
  const [rightPanelEditing, setRightPanelEditing] = useState(false);
  const [leftExpanded, setLeftExpanded] = useState(false);
  const [editingRequest, setEditingRequest] = useState<SavedRequest | undefined>();
  const [lastEditedEndpointId, setLastEditedEndpointId] = useState<string | null>(null);

  // Group endpoints by tag
  const endpointsByTag = useMemo(() => {
    if (!spec) return new Map();
    return getEndpointsByTag(spec.endpoints);
  }, [spec]);

  // Get all tags including custom ones
  const allTags = useMemo(() => {
    if (!spec) return [];
    return savedRequests.getAllTags(spec.tags);
  }, [spec, savedRequests]);

  // Panel navigation - now self-contained
  const panelNav = usePanelNavigation({
    allTags,
    endpointsByTag,
    savedRequestsByTag: savedRequests.getRequestsByTag,
    enabled: mode === 'browse' && !showInfoPopup && !showHelpPopup,
    onQuit: exit,
    onReload: reload,
  });

  // Get derived state from the navigation hook
  const { flatList, selectedItem } = panelNav;

  // Clear response and load overrides when selection changes
  useEffect(() => {
    request.clear();
    setShowResetConfirm(false);

    // Load overrides for the selected endpoint
    if (selectedItem?.type === 'endpoint') {
      const endpoint = selectedItem.endpoint!;
      const override = overrides.getOverride(endpoint.method, endpoint.path);
      if (override) {
        setParameterValues(override.params);
        setCustomParams(override.customParams || []);
        setDisabledParams(override.disabledParams || []);
        setBody(override.body || '');
        setOverridePath(override.overridePath);
        setOverrideMethod(override.overrideMethod);
      } else {
        setParameterValues({});
        setCustomParams([]);
        setDisabledParams([]);
        setBody('');
        setOverridePath(undefined);
        setOverrideMethod(undefined);
      }
      setLastEditedEndpointId(selectedItem.id);
    } else {
      setParameterValues({});
      setCustomParams([]);
      setDisabledParams([]);
      setBody('');
      setOverridePath(undefined);
      setOverrideMethod(undefined);
      setLastEditedEndpointId(null);
    }
  }, [selectedItem?.id]);

  // Tag counts for display
  const tagCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const tag of allTags) {
      const endpoints = endpointsByTag.get(tag) || [];
      const saved = savedRequests.getRequestsByTag(tag);
      counts.set(tag, endpoints.length + saved.length);
    }
    return counts;
  }, [allTags, endpointsByTag, savedRequests]);

  // Handle additional keyboard input for modes
  useInput(
    (input, key) => {
      if (mode !== 'browse') return;

      // Info popup
      if (input === 'i') {
        setShowInfoPopup(prev => !prev);
        return;
      }

      // Help popup
      if (input === '?') {
        setShowHelpPopup(prev => !prev);
        return;
      }

      // Toggle left panel expand
      if (input === '[') {
        setLeftExpanded(prev => !prev);
        return;
      }

      // Open manual request mode
      if (input === 'm') {
        setMode('manual');
        setEditingRequest(undefined);
        return;
      }

      // Try it out mode
      if (input === 't' && (selectedItem?.type === 'endpoint' || selectedItem?.type === 'savedRequest')) {
        if (!body && selectedItem.type === 'endpoint' && spec) {
          const op = selectedItem.endpoint!.operation;
          const schema = op.requestBody?.content?.['application/json']?.schema;
          if (schema) {
            const components = spec.spec.components as Record<string, unknown> | undefined;
            const scaffolded = scaffoldBody(schema as Record<string, unknown>, components);
            if (scaffolded !== null) setBody(JSON.stringify(scaffolded, null, 2));
          }
        }
        setMode('tryit');
        panelNav.setRightMode('tryItOut');
        panelNav.setActivePanel('right');
        panelNav.setRightScroll(0);
        return;
      }

      // Execute request
      if (input === 'e' && (selectedItem?.type === 'endpoint' || selectedItem?.type === 'savedRequest')) {
        executeCurrentEndpoint();
        return;
      }
    },
    { isActive: mode === 'browse' && !showInfoPopup && !showHelpPopup }
  );

  // Handle try-it-out mode input
  useInput(
    (input, key) => {
      if (mode !== 'tryit') return;

      if (key.escape && !rightPanelEditing) {
        // Save overrides before exiting
        if (selectedItem?.type === 'endpoint') {
          const endpoint = selectedItem.endpoint!;
          overrides.saveOverride(
            endpoint.method,
            endpoint.path,
            parameterValues,
            customParams,
            disabledParams,
            body || undefined,
            overridePath,
            overrideMethod
          );
        }
        setShowResetConfirm(false);
        setMode('browse');
        panelNav.setRightMode('details');
        return;
      }
    },
    { isActive: mode === 'tryit' }
  );

  const executeCurrentEndpoint = async () => {
    if (!spec || !selectedItem) return;

    const servers = spec.spec.servers || [{ url: 'http://localhost' }];
    const baseUrl = servers[selectedServer]?.url || servers[0].url;

    const envVars = activeEnvVarsRef.current;

    if (selectedItem.type === 'endpoint') {
      const endpoint = selectedItem.endpoint!;

      // Save overrides before executing
      await overrides.saveOverride(
        endpoint.method,
        endpoint.path,
        parameterValues,
        customParams,
        disabledParams,
        body || undefined,
        overridePath,
        overrideMethod
      );

      // Build query params from parameter values (excluding disabled)
      const queryParams: KeyValuePair[] = [];
      const headerParams: KeyValuePair[] = [];

      for (const param of endpoint.operation.parameters || []) {
        // Skip disabled params
        if (disabledParams.includes(param.name)) continue;

        const value = interpolate(parameterValues[param.name] || '', envVars);
        if (param.in === 'query' && value) {
          queryParams.push({
            id: param.name,
            key: param.name,
            value,
            enabled: true,
          });
        } else if (param.in === 'header' && value) {
          headerParams.push({
            id: param.name,
            key: param.name,
            value,
            enabled: true,
          });
        }
      }

      // Add custom params
      for (const param of customParams) {
        if (!param.enabled || !param.name) continue;

        if (param.in === 'query') {
          queryParams.push({
            id: param.id,
            key: param.name,
            value: interpolate(param.value, envVars),
            enabled: true,
          });
        } else if (param.in === 'header') {
          headerParams.push({
            id: param.id,
            key: param.name,
            value: interpolate(param.value, envVars),
            enabled: true,
          });
        }
        // Note: custom path params are handled below
      }

      // Build path with path params (excluding disabled)
      // Use override path if set, otherwise use spec path
      let path = overridePath || endpoint.path;
      for (const param of endpoint.operation.parameters || []) {
        if (param.in === 'path' && !disabledParams.includes(param.name)) {
          const value = interpolate(parameterValues[param.name] || '', envVars);
          path = path.replace(`{${param.name}}`, encodeURIComponent(value));
        }
      }

      // Apply custom path params (replace {name} patterns)
      for (const param of customParams) {
        if (param.enabled && param.in === 'path' && param.name) {
          path = path.replace(`{${param.name}}`, encodeURIComponent(interpolate(param.value, envVars)));
        }
      }

      // Use override method if set, otherwise use spec method
      const method = overrideMethod || endpoint.method;

      await request.execute(
        method, baseUrl, path, queryParams, headerParams, body ? interpolate(body, envVars) : undefined,
        endpoint.operation.security || spec.spec.security,
        spec.spec.components?.securitySchemes as Record<string, SecuritySchemeObject> | undefined,
        auth.store.credentials
      );
    } else if (selectedItem.type === 'savedRequest') {
      const saved = selectedItem.savedRequest!;
      await request.execute(
        saved.method,
        baseUrl,
        saved.path,
        saved.queryParams.map(p => ({ ...p, value: interpolate(p.value, envVars) })),
        saved.headers.map(h => ({ ...h, value: interpolate(h.value, envVars) })),
        saved.body ? interpolate(saved.body, envVars) : undefined
      );
    }
  };

  const handleManualExecute = async (
    method: string,
    path: string,
    queryParams: KeyValuePair[],
    headers: KeyValuePair[],
    body?: string
  ) => {
    if (!spec) return;

    const servers = spec.spec.servers || [{ url: 'http://localhost' }];
    const baseUrl = servers[selectedServer]?.url || servers[0].url;

    await request.execute(method, baseUrl, path, queryParams, headers, body);
  };

  const handleManualSave = async (
    requestData: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'>
  ) => {
    // Check if tag is custom and doesn't exist
    if (!spec?.tags.includes(requestData.tag)) {
      await savedRequests.createTag({ name: requestData.tag });
    }

    if (editingRequest) {
      await savedRequests.update(editingRequest.id, requestData);
    } else {
      await savedRequests.save(requestData);
    }

    setMode('browse');
    setEditingRequest(undefined);
  };

  const handleManualClose = () => {
    setMode('browse');
    setEditingRequest(undefined);
    request.clear();
  };

  const handleParameterChange = useCallback((name: string, value: string) => {
    setParameterValues(prev => ({ ...prev, [name]: value }));
  }, []);

  const handleResetOverrides = useCallback(() => {
    setShowResetConfirm(true);
  }, []);

  const handleResetConfirmResponse = useCallback(async (confirmed: boolean) => {
    setShowResetConfirm(false);
    if (confirmed && selectedItem?.type === 'endpoint') {
      const endpoint = selectedItem.endpoint!;
      // Delete the override from storage
      await overrides.deleteOverride(endpoint.method, endpoint.path);
      // Reset local state to defaults
      setParameterValues({});
      setCustomParams([]);
      setDisabledParams([]);
      setBody('');
      setOverridePath(undefined);
      setOverrideMethod(undefined);
    }
  }, [selectedItem, overrides]);

  // Loading state
  if (loading) {
    return (
      <Box padding={2}>
        <Spinner label="Loading OpenAPI specification..." />
      </Box>
    );
  }

  // Error state
  if (error || !spec) {
    return (
      <Box flexDirection="column" padding={2}>
        <Text color="red" bold>
          Error loading OpenAPI specification
        </Text>
        <Text color="red">{error || 'Unknown error'}</Text>
        <Box marginTop={1}>
          <Text dimColor>Source: {source}</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Press Ctrl+R to retry or q to quit</Text>
        </Box>
      </Box>
    );
  }

  // Manual request mode
  if (mode === 'manual') {
    return (
      <Box flexDirection="column" flexGrow={1}>
        <Box flexGrow={1}>
          <ManualRequest
            availableTags={allTags}
            editingRequest={editingRequest}
            onExecute={handleManualExecute}
            onSave={handleManualSave}
            onClose={handleManualClose}
          />
        </Box>
        {request.response && (
          <Box flexDirection="column" marginTop={1} borderStyle="single" borderColor="gray" paddingX={1}>
            <Text bold>Response: </Text>
            <Text color={request.response.status < 300 ? 'green' : 'red'}>
              {request.response.status} {request.response.statusText}
            </Text>
            <Text>{request.response.body.slice(0, 200)}...</Text>
          </Box>
        )}
        {request.loading && (
          <Box padding={1}>
            <Spinner label="Executing request..." />
          </Box>
        )}
        <StatusBar mode="manual" />
      </Box>
    );
  }

  // Main two-panel view
  const leftWidthPct = leftExpanded ? 50 : 30;

  return (
    <Box flexDirection="column" flexGrow={1}>
      <Header
        spec={spec.spec}
        selectedServer={selectedServer}
        onServerChange={setSelectedServer}
        collectionName={collectionName}
        activeEnvName={envs.activeEnv?.name}
      />
      {/* Content area — popup replaces panels when open */}
      {showHelpPopup ? (
        <Box flexGrow={1} overflow="hidden">
          <HelpPopup onClose={() => setShowHelpPopup(false)} height={contentHeight} />
        </Box>
      ) : showInfoPopup ? (
        <Box flexDirection="column" flexGrow={1}>
          <InfoPopup
            spec={spec.spec}
            selectedServer={selectedServer}
            onServerChange={setSelectedServer}
            onClose={() => setShowInfoPopup(false)}
            collectionName={collectionName}
            credentials={auth.store.credentials}
            onSetCredential={auth.setCredential}
            environments={envs.store}
            onAddEnvironment={envs.addEnvironment}
            onDeleteEnvironment={envs.deleteEnvironment}
            onSetActive={envs.setActive}
            onSetVariable={envs.setVariable}
            onDeleteVariable={envs.deleteVariable}
          />
          <Box flexGrow={1} />
        </Box>
      ) : (
        <Box flexGrow={1} overflow="hidden">
          <LeftPanel
            items={flatList}
            selectedIndex={panelNav.leftIndex}
            expandedTags={panelNav.expandedTags}
            isActive={panelNav.activePanel === 'left' && mode === 'browse'}
            tagCounts={tagCounts}
            height={contentHeight}
            widthPct={leftWidthPct}
            title={spec.spec.info.title}
            hasPathMethodOverride={overrides.hasPathMethodOverride}
            hasBodyOverride={overrides.hasBodyOverride}
            hasParamsOverride={overrides.hasParamsOverride}
          />
          <RightPanel
            selectedItem={selectedItem}
            mode={mode === 'tryit' ? 'tryItOut' : panelNav.rightMode}
            isActive={panelNav.activePanel === 'right' || mode === 'tryit'}
            scrollOffset={panelNav.rightScroll}
            parameterValues={parameterValues}
            onParameterChange={handleParameterChange}
            body={body}
            onBodyChange={setBody}
            response={request.response}
            curl={request.curl || undefined}
            isLoading={request.loading}
            height={contentHeight}
            onScrollChange={panelNav.setRightScroll}
            customParams={customParams}
            onCustomParamsChange={setCustomParams}
            disabledParams={disabledParams}
            onDisabledParamsChange={setDisabledParams}
            overridePath={overridePath}
            overrideMethod={overrideMethod}
            onOverridePathChange={setOverridePath}
            onOverrideMethodChange={setOverrideMethod}
            onResetOverrides={handleResetOverrides}
            showResetConfirm={showResetConfirm}
            onResetConfirmResponse={handleResetConfirmResponse}
            onEditingChange={setRightPanelEditing}
            specComponents={spec.spec.components as Record<string, unknown> | undefined}
          />
        </Box>
      )}

      <StatusBar
        mode={mode}
        activePanel={panelNav.activePanel}
        position={`${panelNav.leftIndex + 1}/${flatList.length}`}
      />
    </Box>
  );
}

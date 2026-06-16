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
  ManualSaveDialog,
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
  const [leftExpanded, setLeftExpanded] = useState(false);
  const [editingRequest, setEditingRequest] = useState<SavedRequest | undefined>();
  const [lastEditedEndpointId, setLastEditedEndpointId] = useState<string | null>(null);

  // Manual request mode state (isolated from spec tryit state)
  const [manualOverridePath, setManualOverridePath] = useState('');
  const [manualOverrideMethod, setManualOverrideMethod] = useState('GET');
  const [manualCustomParams, setManualCustomParams] = useState<CustomParameter[]>([]);
  const [manualBody, setManualBody] = useState('');
  const [showManualSaveDialog, setShowManualSaveDialog] = useState(false);
  const [rightPanelNormalMode, setRightPanelNormalMode] = useState(true);

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

  // When response arrives, scroll back to top so params stay visible above response
  useEffect(() => {
    if (request.response) {
      panelNav.setRightScroll(0);
    }
  }, [request.response]);

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

      // Open manual request mode (new empty request)
      if (input === 'm') {
        setManualOverridePath('');
        setManualOverrideMethod('GET');
        setManualCustomParams([]);
        setManualBody('');
        setEditingRequest(undefined);
        setShowManualSaveDialog(false);
        setMode('manual');
        panelNav.setActivePanel('right');
        return;
      }

      // Edit saved request (E = open in manual mode pre-filled)
      if (input === 'E' && selectedItem?.type === 'savedRequest') {
        const saved = selectedItem.savedRequest!;
        setManualOverridePath(saved.path);
        setManualOverrideMethod(saved.method);
        setManualCustomParams([
          ...saved.queryParams.map(p => ({ id: p.id, name: p.key, value: p.value, in: 'query' as const, enabled: p.enabled })),
          ...saved.headers.map(h => ({ id: h.id, name: h.key, value: h.value, in: 'header' as const, enabled: h.enabled })),
        ]);
        setManualBody(saved.body || '');
        setEditingRequest(saved);
        setShowManualSaveDialog(false);
        request.clear();
        setMode('manual');
        panelNav.setActivePanel('right');
        panelNav.setRightScroll(0);
        return;
      }

      // Delete saved request
      if (input === 'D' && selectedItem?.type === 'savedRequest') {
        void savedRequests.remove(selectedItem.savedRequest!.id);
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
        panelNav.setRightScroll(0);
        executeCurrentEndpoint();
        return;
      }
    },
    { isActive: mode === 'browse' && !showInfoPopup && !showHelpPopup }
  );

  // Handle try-it-out and manual mode input
  useInput(
    (input, key) => {
      const isManual = mode === 'manual';
      const isTryit = mode === 'tryit';
      if (!isManual && !isTryit) return;

      if (key.escape && rightPanelNormalMode) {
        if (isManual) {
          setMode('browse');
          setEditingRequest(undefined);
          setShowManualSaveDialog(false);
          request.clear();
          return;
        }
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

      // Save (manual only)
      if (input === 's' && isManual && rightPanelNormalMode) {
        setShowManualSaveDialog(true);
        return;
      }

      // Execute
      if (input === 'e' && rightPanelNormalMode) {
        panelNav.setRightScroll(0);
        if (isManual) {
          handleManualExecuteFromState();
        } else {
          executeCurrentEndpoint();
        }
        return;
      }

      // Delete editing request (manual only)
      if (input === 'd' && isManual && editingRequest && rightPanelNormalMode) {
        void savedRequests.remove(editingRequest.id).then(() => {
          setMode('browse');
          setEditingRequest(undefined);
        });
        return;
      }
    },
    { isActive: mode === 'tryit' || mode === 'manual' }
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

  const handleManualExecuteFromState = async () => {
    if (!manualOverridePath) return;
    const envVars = activeEnvVarsRef.current;
    const servers = spec?.spec.servers || [{ url: 'http://localhost' }];
    const baseUrl = servers[selectedServer]?.url || servers[0].url;

    const queryParams: KeyValuePair[] = manualCustomParams
      .filter(p => p.in === 'query' && p.enabled && p.name)
      .map(p => ({ id: p.id, key: p.name, value: interpolate(p.value, envVars), enabled: true }));
    const headerParams: KeyValuePair[] = manualCustomParams
      .filter(p => p.in === 'header' && p.enabled && p.name)
      .map(p => ({ id: p.id, key: p.name, value: interpolate(p.value, envVars), enabled: true }));

    await request.execute(
      manualOverrideMethod || 'GET',
      baseUrl,
      interpolate(manualOverridePath, envVars),
      queryParams,
      headerParams,
      manualBody ? interpolate(manualBody, envVars) : undefined
    );
  };

  const handleManualSaveFromDialog = async (name: string, tag: string) => {
    setShowManualSaveDialog(false);

    if (!spec?.tags.includes(tag)) {
      await savedRequests.createTag({ name: tag });
    }

    const queryParams: KeyValuePair[] = manualCustomParams
      .filter(p => p.in === 'query')
      .map(p => ({ id: p.id, key: p.name, value: p.value, enabled: p.enabled }));
    const headers: KeyValuePair[] = manualCustomParams
      .filter(p => p.in === 'header')
      .map(p => ({ id: p.id, key: p.name, value: p.value, enabled: p.enabled }));

    const requestData: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'> = {
      method: (manualOverrideMethod || 'GET') as SavedRequest['method'],
      path: manualOverridePath,
      queryParams,
      headers,
      body: manualBody,
      bodyType: 'json',
      name,
      tag,
    };

    if (editingRequest) {
      await savedRequests.update(editingRequest.id, requestData);
    } else {
      await savedRequests.save(requestData);
    }

    setMode('browse');
    setEditingRequest(undefined);
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
      ) : showManualSaveDialog ? (
        <Box flexGrow={1} alignItems="center" justifyContent="center">
          <ManualSaveDialog
            availableTags={allTags}
            initialName={editingRequest?.name}
            initialTag={editingRequest?.tag}
            onSave={handleManualSaveFromDialog}
            onCancel={() => setShowManualSaveDialog(false)}
          />
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
            selectedItem={mode === 'manual' ? null : selectedItem}
            mode={mode === 'tryit' || mode === 'manual' ? 'tryItOut' : panelNav.rightMode}
            isActive={panelNav.activePanel === 'right' || mode === 'tryit' || mode === 'manual'}
            scrollOffset={panelNav.rightScroll}
            parameterValues={mode === 'manual' ? {} : parameterValues}
            onParameterChange={mode === 'manual' ? () => {} : handleParameterChange}
            body={mode === 'manual' ? manualBody : body}
            onBodyChange={mode === 'manual' ? setManualBody : setBody}
            response={request.response}
            curl={request.curl || undefined}
            isLoading={request.loading}
            height={contentHeight}
            onScrollChange={panelNav.setRightScroll}
            customParams={mode === 'manual' ? manualCustomParams : customParams}
            onCustomParamsChange={mode === 'manual' ? setManualCustomParams : setCustomParams}
            disabledParams={mode === 'manual' ? [] : disabledParams}
            onDisabledParamsChange={mode === 'manual' ? undefined : setDisabledParams}
            overridePath={mode === 'manual' ? manualOverridePath : overridePath}
            overrideMethod={mode === 'manual' ? manualOverrideMethod : overrideMethod}
            onOverridePathChange={mode === 'manual' ? setManualOverridePath : setOverridePath}
            onOverrideMethodChange={mode === 'manual' ? setManualOverrideMethod : setOverrideMethod}
            onResetOverrides={handleResetOverrides}
            showResetConfirm={showResetConfirm}
            onResetConfirmResponse={handleResetConfirmResponse}

            onNormalModeChange={setRightPanelNormalMode}
            manualMode={mode === 'manual'}
            editingRequest={mode === 'manual' ? editingRequest : undefined}
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

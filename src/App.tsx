import React, { useState, useMemo, useCallback, useEffect } from 'react';
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
  ManualRequestPanel,
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
  useAppKeyboard,
} from './hooks/index.js';
import { getEndpointsByTag } from './utils/parser.js';
import { interpolate } from './utils/interpolate.js';
import { ParameterCollector } from './utils/parameterCollector.js';
import type { KeyValuePair, SavedRequest, CustomParameter, SecuritySchemeObject, RequestSpec } from './types/index.js';

interface AppProps {
  source: string;
  collectionName?: string;
}

type ManualState = {
  mode: 'manual';
  path: string;
  method: string;
  customParams: CustomParameter[];
  body: string;
  editingRequest?: SavedRequest;
  showSaveDialog: boolean;
};

type RenameTagState = {
  mode: 'renameTag';
  tagName: string;
  value: string;
};

type AppState = { mode: 'browse' | 'tryit' } | ManualState | RenameTagState;

export function App({ source, collectionName }: AppProps) {
  const { exit } = useApp();
  const { height: terminalHeight } = useScreenSize();
  const { spec, loading, error, reload } = useOpenAPI(source);
  const savedRequests = useSavedRequests();
  const request = useRequest();
  const overrides = useOverrides();
  const auth = useAuth();
  const envs = useEnvironments();

  const contentHeight = Math.max(terminalHeight - 6, 10);

  const [appState, setAppState] = useState<AppState>({ mode: 'browse' });
  const [selectedServer, setSelectedServer] = useState(0);

  // Endpoint editing state — shared between browse (read-only) and tryit (editable)
  const [parameterValues, setParameterValues] = useState<Record<string, string>>({});
  const [customParams, setCustomParams] = useState<CustomParameter[]>([]);
  const [disabledParams, setDisabledParams] = useState<string[]>([]);
  const [body, setBody] = useState('');
  const [overridePath, setOverridePath] = useState<string | undefined>();
  const [overrideMethod, setOverrideMethod] = useState<string | undefined>();
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [tagDeleteConfirm, setTagDeleteConfirm] = useState<string | null>(null);

  const [showInfoPopup, setShowInfoPopup] = useState(false);
  const [showHelpPopup, setShowHelpPopup] = useState(false);
  const [leftExpanded, setLeftExpanded] = useState(false);
  const [lastEditedEndpointId, setLastEditedEndpointId] = useState<string | null>(null);
  const [rightPanelNormalMode, setRightPanelNormalMode] = useState(true);

  const endpointsByTag = useMemo(() => {
    if (!spec) return new Map();
    return getEndpointsByTag(spec.endpoints);
  }, [spec]);

  const allTags = useMemo(() => {
    if (!spec) return [];
    return savedRequests.getAllTags(spec.tags);
  }, [spec, savedRequests]);

  const panelNav = usePanelNavigation({
    allTags,
    endpointsByTag,
    savedRequestsByTag: savedRequests.getRequestsByTag,
    enabled: appState.mode === 'browse' && !showInfoPopup && !showHelpPopup,
    onQuit: exit,
    onReload: reload,
  });

  const { flatList, selectedItem } = panelNav;

  useEffect(() => {
    if (request.response) panelNav.setRightScroll(0);
  }, [request.response]);

  useEffect(() => {
    request.clear();
    setShowResetConfirm(false);

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

  const tagCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const tag of allTags) {
      const endpoints = endpointsByTag.get(tag) || [];
      const saved = savedRequests.getRequestsByTag(tag);
      counts.set(tag, endpoints.length + saved.length);
    }
    return counts;
  }, [allTags, endpointsByTag, savedRequests]);

  // Browse-mode popup/layout keys (remain in App — they touch local UI state)
  useInput(
    (input, _key) => {
      if (appState.mode !== 'browse') return;
      if (input === 'i') { setShowInfoPopup(prev => !prev); return; }
      if (input === '?') { setShowHelpPopup(prev => !prev); return; }
      if (input === '[') { setLeftExpanded(prev => !prev); return; }
    },
    { isActive: appState.mode === 'browse' && !showInfoPopup && !showHelpPopup }
  );

  const executeCurrentEndpoint = async () => {
    if (!spec || !selectedItem) return;

    const servers = spec.spec.servers || [{ url: 'http://localhost' }];
    const baseUrl = servers[selectedServer]?.url || servers[0].url;
    const envVars = envs.activeEnv?.variables;

    if (selectedItem.type === 'endpoint') {
      const endpoint = selectedItem.endpoint!;

      await overrides.saveOverride(
        endpoint.method, endpoint.path, parameterValues,
        customParams, disabledParams, body || undefined, overridePath, overrideMethod
      );

      const collector = new ParameterCollector(
        endpoint.operation.parameters || [], customParams, disabledParams, parameterValues, envVars
      );

      const spec_: RequestSpec = {
        method: overrideMethod || endpoint.method,
        baseUrl,
        path: collector.applyPathParams(overridePath || endpoint.path),
        queryParams: collector.getQueryParams(),
        headerParams: collector.getHeaderParams(),
        body: body ? interpolate(body, envVars) : undefined,
        operationSecurity: endpoint.operation.security || spec.spec.security,
        securitySchemes: spec.spec.components?.securitySchemes as Record<string, SecuritySchemeObject> | undefined,
        authCredentials: auth.store.credentials,
      };

      await request.execute(spec_);
    } else if (selectedItem.type === 'savedRequest') {
      const saved = selectedItem.savedRequest!;
      const spec_: RequestSpec = {
        method: saved.method,
        baseUrl,
        path: interpolate(saved.path, envVars),
        queryParams: saved.queryParams.map(p => ({ ...p, value: interpolate(p.value, envVars) })),
        headerParams: saved.headers.map(h => ({ ...h, value: interpolate(h.value, envVars) })),
        body: saved.body ? interpolate(saved.body, envVars) : undefined,
        operationSecurity: spec.spec.security,
        securitySchemes: spec.spec.components?.securitySchemes as Record<string, SecuritySchemeObject> | undefined,
        authCredentials: auth.store.credentials,
      };
      await request.execute(spec_);
    }
  };

  const handleManualExecuteFromState = async () => {
    const manual = appState as ManualState;
    if (!manual.path) return;

    const envVars = envs.activeEnv?.variables;
    const servers = spec?.spec.servers || [{ url: 'http://localhost' }];
    const baseUrl = servers[selectedServer]?.url || servers[0].url;

    const collector = new ParameterCollector([], manual.customParams, [], {}, envVars);
    const spec_: RequestSpec = {
      method: manual.method || 'GET',
      baseUrl,
      path: interpolate(manual.path, envVars),
      queryParams: collector.getQueryParams(),
      headerParams: collector.getHeaderParams(),
      body: manual.body ? interpolate(manual.body, envVars) : undefined,
      operationSecurity: spec?.spec.security,
      securitySchemes: spec?.spec.components?.securitySchemes as Record<string, SecuritySchemeObject> | undefined,
      authCredentials: auth.store.credentials,
    };
    await request.execute(spec_);
  };

  useAppKeyboard({
    appState,
    selectedItem,
    spec,
    body,
    parameterValues,
    customParams,
    disabledParams,
    overridePath,
    overrideMethod,
    rightPanelNormalMode,
    panelNav,
    savedRequests,
    request,
    overrides,
    setAppState,
    setBody,
    setShowResetConfirm,
    executeCurrentEndpoint,
    handleManualExecuteFromState,
    tagDeleteConfirm,
    setTagDeleteConfirm,
  });

  const handleRenameTagSubmit = useCallback(async (tagName: string, value: string) => {
    const newName = value.trim();
    if (newName && newName !== tagName) {
      await savedRequests.renameTag(tagName, newName);
    }
    setAppState({ mode: 'browse' });
  }, [savedRequests]);

  const handleManualSaveFromDialog = async (name: string, tag: string) => {
    const manual = appState as ManualState;

    if (tag !== 'default' && !spec?.tags.includes(tag)) {
      await savedRequests.createTag({ name: tag });
    }

    const queryParams: KeyValuePair[] = manual.customParams
      .filter(p => p.in === 'query')
      .map(p => ({ id: p.id, key: p.name, value: p.value, enabled: p.enabled }));
    const headers: KeyValuePair[] = manual.customParams
      .filter(p => p.in === 'header')
      .map(p => ({ id: p.id, key: p.name, value: p.value, enabled: p.enabled }));

    const requestData: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'> = {
      method: (manual.method || 'GET') as SavedRequest['method'],
      path: manual.path,
      queryParams,
      headers,
      body: manual.body,
      bodyType: 'json',
      name,
      tag,
    };

    if (manual.editingRequest) {
      await savedRequests.update(manual.editingRequest.id, requestData);
    } else {
      await savedRequests.save(requestData);
    }

    setAppState({ mode: 'browse' });
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
      await overrides.deleteOverride(endpoint.method, endpoint.path);
      setParameterValues({});
      setCustomParams([]);
      setDisabledParams([]);
      setBody('');
      setOverridePath(undefined);
      setOverrideMethod(undefined);
    }
  }, [selectedItem, overrides]);

  if (loading) {
    return (
      <Box padding={2}>
        <Spinner label="Loading OpenAPI specification..." />
      </Box>
    );
  }

  if (error || !spec) {
    return (
      <Box flexDirection="column" padding={2}>
        <Text color="red" bold>Error loading OpenAPI specification</Text>
        <Text color="red">{error || 'Unknown error'}</Text>
        <Box marginTop={1}><Text dimColor>Source: {source}</Text></Box>
        <Box marginTop={1}><Text dimColor>Press Ctrl+R to retry or q to quit</Text></Box>
      </Box>
    );
  }

  const leftWidthPct = leftExpanded ? 50 : 30;
  const isManual = appState.mode === 'manual';
  const manual = isManual ? (appState as ManualState) : null;

  return (
    <Box flexDirection="column" flexGrow={1}>
      <Header
        spec={spec.spec}
        selectedServer={selectedServer}
        onServerChange={setSelectedServer}
        collectionName={collectionName}
        activeEnvName={envs.activeEnv?.name}
      />
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
      ) : manual?.showSaveDialog ? (
        <Box flexGrow={1} alignItems="center" justifyContent="center">
          <ManualSaveDialog
            availableTags={allTags}
            initialName={manual.editingRequest?.name}
            initialTag={manual.editingRequest?.tag}
            onSave={handleManualSaveFromDialog}
            onCancel={() => setAppState({ ...manual, showSaveDialog: false })}
          />
        </Box>
      ) : (
        <Box flexGrow={1} overflow="hidden">
          <LeftPanel
            items={flatList}
            selectedIndex={panelNav.leftIndex}
            expandedTags={panelNav.expandedTags}
            isActive={panelNav.activePanel === 'left' && appState.mode === 'browse'}
            tagCounts={tagCounts}
            height={contentHeight}
            widthPct={leftWidthPct}
            title={spec.spec.info.title}
            hasPathMethodOverride={overrides.hasPathMethodOverride}
            hasBodyOverride={overrides.hasBodyOverride}
            hasParamsOverride={overrides.hasParamsOverride}
          />
          {isManual && manual ? (
            <ManualRequestPanel
              path={manual.path}
              method={manual.method}
              customParams={manual.customParams}
              body={manual.body}
              editingRequest={manual.editingRequest}
              isActive={true}
              response={request.response ?? undefined}
              curl={request.curl ?? undefined}
              isLoading={request.loading}
              height={contentHeight}
              scrollOffset={panelNav.rightScroll}
              onPathChange={(path) => setAppState({ ...manual, path })}
              onMethodChange={(method) => setAppState({ ...manual, method })}
              onCustomParamsChange={(customParams) => setAppState({ ...manual, customParams })}
              onBodyChange={(body) => setAppState({ ...manual, body })}
              onNormalModeChange={setRightPanelNormalMode}
              onScrollReset={() => panelNav.setRightScroll(0)}
            />
          ) : (
            <RightPanel
              selectedItem={selectedItem}
              mode={appState.mode === 'tryit' ? 'tryItOut' : panelNav.rightMode}
              isActive={panelNav.activePanel === 'right' || appState.mode === 'tryit'}
              scrollOffset={panelNav.rightScroll}
              parameterValues={parameterValues}
              onParameterChange={handleParameterChange}
              body={body}
              onBodyChange={setBody}
              response={request.response}
              curl={request.curl ?? undefined}
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
              onNormalModeChange={setRightPanelNormalMode}
              specComponents={spec.spec.components as Record<string, unknown> | undefined}
              renamingTag={appState.mode === 'renameTag' ? { tagName: appState.tagName, value: appState.value } : null}
              onRenamingTagChange={(value) => {
                if (appState.mode === 'renameTag') setAppState({ ...appState, value });
              }}
              onRenamingTagSubmit={handleRenameTagSubmit}
              tagDeleteConfirm={tagDeleteConfirm ? { tagName: tagDeleteConfirm, count: savedRequests.getRequestsByTag(tagDeleteConfirm).length } : null}
              customTagNames={savedRequests.customTags.map(t => t.name)}
              tagEndpoints={selectedItem?.type === 'tag' ? (endpointsByTag.get(selectedItem.tagName) || []) : undefined}
              tagSavedRequests={selectedItem?.type === 'tag' ? savedRequests.getRequestsByTag(selectedItem.tagName) : undefined}
            />
          )}
        </Box>
      )}

      <StatusBar
        mode={appState.mode}
        activePanel={panelNav.activePanel}
        position={`${panelNav.leftIndex + 1}/${flatList.length}`}
      />
    </Box>
  );
}

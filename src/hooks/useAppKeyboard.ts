import { useInput } from 'ink';
import { scaffoldBody } from '../utils/scaffoldBody.js';
import type { ParsedSpec } from '../utils/parser.js';
import type { FlatListItem } from './usePanelNavigation.js';
import type { ActivePanel } from './usePanelNavigation.js';
import type { RightPanelMode } from './usePanelNavigation.js';
import type { SavedRequest, CustomParameter, SecuritySchemeObject } from '../types/index.js';

type ManualState = {
  mode: 'manual';
  path: string;
  method: string;
  customParams: CustomParameter[];
  body: string;
  editingRequest?: SavedRequest;
  showSaveDialog: boolean;
};

type AppState = { mode: 'browse' | 'tryit' } | ManualState;

interface PanelNav {
  setActivePanel: (panel: ActivePanel) => void;
  setRightMode: (mode: RightPanelMode) => void;
  setRightScroll: (scroll: number) => void;
}

interface SavedRequestsHook {
  remove: (id: string) => Promise<boolean>;
  createTag: (tag: { name: string }) => Promise<unknown>;
}

interface RequestHook {
  clear: () => void;
}

interface OverridesHook {
  saveOverride: (
    method: string,
    path: string,
    params: Record<string, string>,
    customParams: CustomParameter[],
    disabledParams: string[],
    body?: string,
    overridePath?: string,
    overrideMethod?: string
  ) => Promise<void>;
}

interface UseAppKeyboardOptions {
  appState: AppState;
  selectedItem: FlatListItem | null;
  spec: ParsedSpec | null;
  body: string;
  parameterValues: Record<string, string>;
  customParams: CustomParameter[];
  disabledParams: string[];
  overridePath: string | undefined;
  overrideMethod: string | undefined;
  rightPanelNormalMode: boolean;
  panelNav: PanelNav;
  savedRequests: SavedRequestsHook;
  request: RequestHook;
  overrides: OverridesHook;
  setAppState: (state: AppState) => void;
  setBody: (body: string) => void;
  setShowResetConfirm: (v: boolean) => void;
  executeCurrentEndpoint: () => Promise<void>;
  handleManualExecuteFromState: () => Promise<void>;
}

export function useAppKeyboard({
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
}: UseAppKeyboardOptions) {
  // Browse-mode keyboard handler
  useInput(
    (input, _key) => {
      if (appState.mode !== 'browse') return;

      if (input === 'm') {
        setAppState({ mode: 'manual', path: '', method: 'GET', customParams: [], body: '', showSaveDialog: false });
        panelNav.setActivePanel('right');
        return;
      }

      if (input === 'E' && selectedItem?.type === 'savedRequest') {
        const saved = selectedItem.savedRequest!;
        request.clear();
        setAppState({
          mode: 'manual',
          path: saved.path,
          method: saved.method,
          customParams: [
            ...saved.queryParams.map(p => ({ id: p.id, name: p.key, value: p.value, in: 'query' as const, enabled: p.enabled })),
            ...saved.headers.map(h => ({ id: h.id, name: h.key, value: h.value, in: 'header' as const, enabled: h.enabled })),
          ],
          body: saved.body || '',
          editingRequest: saved,
          showSaveDialog: false,
        });
        panelNav.setActivePanel('right');
        panelNav.setRightScroll(0);
        return;
      }

      if (input === 'D' && selectedItem?.type === 'savedRequest') {
        void savedRequests.remove(selectedItem.savedRequest!.id);
        return;
      }

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
        setAppState({ mode: 'tryit' });
        panelNav.setRightMode('tryItOut');
        panelNav.setActivePanel('right');
        panelNav.setRightScroll(0);
        return;
      }

      if (input === 'e' && (selectedItem?.type === 'endpoint' || selectedItem?.type === 'savedRequest')) {
        panelNav.setRightScroll(0);
        void executeCurrentEndpoint();
        return;
      }
    },
    { isActive: appState.mode === 'browse' }
  );

  // Tryit + manual keyboard handler
  useInput(
    (input, key) => {
      const isManual = appState.mode === 'manual';
      const isTryit = appState.mode === 'tryit';
      if (!isManual && !isTryit) return;

      if (key.escape && rightPanelNormalMode) {
        if (isManual) {
          setAppState({ mode: 'browse' });
          request.clear();
          return;
        }
        if (selectedItem?.type === 'endpoint') {
          const endpoint = selectedItem.endpoint!;
          void overrides.saveOverride(
            endpoint.method, endpoint.path, parameterValues,
            customParams, disabledParams, body || undefined, overridePath, overrideMethod
          );
        }
        setShowResetConfirm(false);
        setAppState({ mode: 'browse' });
        panelNav.setRightMode('details');
        return;
      }

      if (input === 's' && isManual && rightPanelNormalMode) {
        const manual = appState as ManualState;
        setAppState({ ...manual, showSaveDialog: true });
        return;
      }

      if (input === 'e' && rightPanelNormalMode) {
        panelNav.setRightScroll(0);
        if (isManual) void handleManualExecuteFromState();
        else void executeCurrentEndpoint();
        return;
      }

      if (input === 'd' && isManual && rightPanelNormalMode) {
        const manual = appState as ManualState;
        if (manual.editingRequest) {
          void savedRequests.remove(manual.editingRequest.id).then(() => {
            setAppState({ mode: 'browse' });
          });
        }
        return;
      }
    },
    { isActive: appState.mode === 'tryit' || appState.mode === 'manual' }
  );

}

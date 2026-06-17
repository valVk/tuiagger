import { useState, useCallback, useMemo } from 'react';
import { useInput } from 'ink';
import type { ParsedEndpoint } from '../utils/parser.js';
import type { SavedRequest } from '../types/index.js';

export type ActivePanel = 'left' | 'right';
export type RightPanelMode = 'details' | 'tryItOut';

export interface FlatListItem {
  type: 'tag' | 'endpoint' | 'savedRequest';
  id: string;
  tagName: string;
  endpoint?: ParsedEndpoint;
  savedRequest?: SavedRequest;
}

interface PanelNavigationState {
  activePanel: ActivePanel;
  leftIndex: number;
  rightScroll: number;
  rightMode: RightPanelMode;
}

interface UsePanelNavigationProps {
  allTags: string[];
  endpointsByTag: Map<string, ParsedEndpoint[]>;
  savedRequestsByTag: (tag: string) => SavedRequest[];
  enabled: boolean;
  onQuit: () => void;
  onReload: () => void;
}

export function usePanelNavigation({
  allTags,
  endpointsByTag,
  savedRequestsByTag,
  enabled,
  onQuit,
  onReload,
}: UsePanelNavigationProps) {
  const [expandedTags, setExpandedTags] = useState(() => new Set(allTags));

  const onToggleTag = useCallback((tagName: string) => {
    setExpandedTags(prev => {
      const next = new Set(prev);
      if (next.has(tagName)) {
        next.delete(tagName);
      } else {
        next.add(tagName);
      }
      return next;
    });
  }, []);

  const onExpandAll = useCallback(() => {
    setExpandedTags(new Set(allTags));
  }, [allTags]);

  const onCollapseAll = useCallback(() => {
    setExpandedTags(new Set());
  }, []);

  const flatList = useMemo(
    () => buildFlatList(allTags, endpointsByTag, savedRequestsByTag, expandedTags),
    [allTags, endpointsByTag, savedRequestsByTag, expandedTags]
  );

  const [state, setState] = useState<PanelNavigationState>({
    activePanel: 'left',
    leftIndex: 0,
    rightScroll: 0,
    rightMode: 'details',
  });

  // Ensure leftIndex is within bounds after list changes
  const safeLeftIndex = Math.min(state.leftIndex, Math.max(0, flatList.length - 1));
  const selectedItem = flatList[safeLeftIndex] || null;

  useInput(
    (input, key) => {
      if (!enabled) return;

      // Quit
      if (input === 'q') {
        onQuit();
        return;
      }

      // Reload
      if (key.ctrl && input === 'r') {
        onReload();
        return;
      }

      // Panel switching
      if (input === 'h' || key.leftArrow) {
        setState(s => ({ ...s, activePanel: 'left', leftIndex: safeLeftIndex }));
        return;
      }
      if (input === 'l' || key.rightArrow) {
        setState(s => ({ ...s, activePanel: 'right', leftIndex: safeLeftIndex }));
        return;
      }

      if (state.activePanel === 'left') {
        // Left panel navigation
        if (input === 'j' || key.downArrow) {
          setState(s => ({
            ...s,
            leftIndex: Math.min(safeLeftIndex + 1, flatList.length - 1),
            rightScroll: 0,
          }));
          return;
        }
        if (input === 'k' || key.upArrow) {
          setState(s => ({
            ...s,
            leftIndex: Math.max(safeLeftIndex - 1, 0),
            rightScroll: 0,
          }));
          return;
        }
        if (input === 'g') {
          setState(s => ({ ...s, leftIndex: 0, rightScroll: 0 }));
          return;
        }
        if (input === 'G') {
          setState(s => ({
            ...s,
            leftIndex: Math.max(0, flatList.length - 1),
            rightScroll: 0,
          }));
          return;
        }

        // Toggle tag expansion
        if (key.return && selectedItem?.type === 'tag') {
          onToggleTag(selectedItem.tagName);
          return;
        }

        // Collapse all
        if (input === 'c') {
          onCollapseAll();
          // Reset index to 0 when collapsing all
          setState(s => ({ ...s, leftIndex: 0 }));
          return;
        }

        // Expand all
        if (input === 'x') {
          onExpandAll();
          return;
        }
      } else {
        // Right panel scroll
        if (input === 'j' || key.downArrow) {
          setState(s => ({ ...s, rightScroll: s.rightScroll + 1 }));
          return;
        }
        if (input === 'k' || key.upArrow) {
          setState(s => ({
            ...s,
            rightScroll: Math.max(0, s.rightScroll - 1),
          }));
          return;
        }
        if (input === 'g') {
          setState(s => ({ ...s, rightScroll: 0 }));
          return;
        }
      }

      // Try it out mode toggle (works from both panels when endpoint selected)
      if (input === 't' && (selectedItem?.type === 'endpoint' || selectedItem?.type === 'savedRequest')) {
        setState(s => ({
          ...s,
          rightMode: s.rightMode === 'tryItOut' ? 'details' : 'tryItOut',
          activePanel: 'right',
        }));
        return;
      }

      // Escape to go back to details
      if (key.escape) {
        setState(s => ({
          ...s,
          rightMode: 'details',
        }));
        return;
      }
    },
    { isActive: enabled }
  );

  const setRightMode = useCallback((mode: RightPanelMode) => {
    setState(s => ({ ...s, rightMode: mode }));
  }, []);

  const setActivePanel = useCallback((panel: ActivePanel) => {
    setState(s => ({ ...s, activePanel: panel }));
  }, []);

  const setRightScroll = useCallback((scroll: number) => {
    setState(s => ({ ...s, rightScroll: scroll }));
  }, []);

  return {
    activePanel: state.activePanel,
    leftIndex: safeLeftIndex,
    rightScroll: state.rightScroll,
    rightMode: state.rightMode,
    expandedTags,
    selectedItem,
    flatList,
    setRightMode,
    setActivePanel,
    setRightScroll,
  };
}

export function buildFlatList(
  tags: string[],
  endpointsByTag: Map<string, ParsedEndpoint[]>,
  savedRequestsByTag: (tag: string) => SavedRequest[],
  expandedTags: Set<string>
): FlatListItem[] {
  const items: FlatListItem[] = [];

  for (const tag of tags) {
    // Add tag item
    items.push({
      type: 'tag',
      id: `tag-${tag}`,
      tagName: tag,
    });

    // Add endpoints if tag is expanded
    if (expandedTags.has(tag)) {
      const endpoints = endpointsByTag.get(tag) || [];
      for (const endpoint of endpoints) {
        items.push({
          type: 'endpoint',
          id: `endpoint-${endpoint.method}-${endpoint.path}`,
          tagName: tag,
          endpoint,
        });
      }

      // Add saved requests
      const savedRequests = savedRequestsByTag(tag);
      for (const savedRequest of savedRequests) {
        items.push({
          type: 'savedRequest',
          id: `saved-${savedRequest.id}`,
          tagName: tag,
          savedRequest,
        });
      }
    }
  }

  return items;
}

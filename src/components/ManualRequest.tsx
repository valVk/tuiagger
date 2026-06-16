import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { KeyValueEditor } from './KeyValueEditor.js';
import type { SavedRequest, KeyValuePair, HttpMethodType } from '../types/index.js';

const HTTP_METHODS: HttpMethodType[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

interface ManualRequestProps {
  availableTags: string[];
  editingRequest?: SavedRequest;
  onExecute: (method: string, path: string, queryParams: KeyValuePair[], headers: KeyValuePair[], body?: string) => void;
  onSave: (request: Omit<SavedRequest, 'id' | 'createdAt' | 'updatedAt'>) => void;
  onClose: () => void;
}

type FocusArea = 'name' | 'tag' | 'newTag' | 'method' | 'path' | 'queryParams' | 'headers' | 'body';

export function ManualRequest({
  availableTags,
  editingRequest,
  onExecute,
  onSave,
  onClose,
}: ManualRequestProps) {
  const [name, setName] = useState(editingRequest?.name || '');
  const [tag, setTag] = useState(editingRequest?.tag || availableTags[0] || 'custom');
  const [newTagMode, setNewTagMode] = useState(false);
  const [newTagName, setNewTagName] = useState('');
  const [methodIndex, setMethodIndex] = useState(
    HTTP_METHODS.indexOf(editingRequest?.method || 'GET')
  );
  const [path, setPath] = useState(editingRequest?.path || '/');
  const [queryParams, setQueryParams] = useState<KeyValuePair[]>(
    editingRequest?.queryParams || [{ id: '1', key: '', value: '', enabled: true }]
  );
  const [headers, setHeaders] = useState<KeyValuePair[]>(
    editingRequest?.headers || [{ id: '1', key: '', value: '', enabled: true }]
  );
  const [body, setBody] = useState(editingRequest?.body || '');

  const [focusArea, setFocusArea] = useState<FocusArea>('name');
  const [kvFocusedIndex, setKvFocusedIndex] = useState(0);
  const [kvFocusedField, setKvFocusedField] = useState<'key' | 'value'>('key');
  const [tagIndex, setTagIndex] = useState(availableTags.indexOf(tag) >= 0 ? availableTags.indexOf(tag) : 0);

  const focusOrder: FocusArea[] = ['name', 'tag', 'method', 'path', 'queryParams', 'headers', 'body'];

  useInput((input, key) => {
    // Close on Escape
    if (key.escape) {
      if (newTagMode) {
        setNewTagMode(false);
        setNewTagName('');
      } else {
        onClose();
      }
      return;
    }

    // Execute with 'e'
    if (input === 'e' && !isTextInputFocused()) {
      handleExecute();
      return;
    }

    // Save with 's'
    if (input === 's' && !isTextInputFocused() && name && (tag || newTagName)) {
      handleSave();
      return;
    }

    // Tab navigation
    if (key.tab) {
      if (key.shift) {
        moveFocusPrev();
      } else {
        moveFocusNext();
      }
      return;
    }

    // Handle key/value editor navigation
    if (focusArea === 'queryParams' || focusArea === 'headers') {
      const pairs = focusArea === 'queryParams' ? queryParams : headers;
      const setPairs = focusArea === 'queryParams' ? setQueryParams : setHeaders;

      if (input === 'a') {
        setPairs([...pairs, { id: Date.now().toString(), key: '', value: '', enabled: true }]);
        setKvFocusedIndex(pairs.length);
        setKvFocusedField('key');
        return;
      }

      if (input === 'd' && pairs.length > 1) {
        setPairs(pairs.filter((_, i) => i !== kvFocusedIndex));
        setKvFocusedIndex(Math.max(0, kvFocusedIndex - 1));
        return;
      }

      if (key.downArrow && kvFocusedIndex < pairs.length - 1) {
        setKvFocusedIndex(kvFocusedIndex + 1);
        return;
      }

      if (key.upArrow && kvFocusedIndex > 0) {
        setKvFocusedIndex(kvFocusedIndex - 1);
        return;
      }

      if (key.leftArrow || key.rightArrow) {
        setKvFocusedField(kvFocusedField === 'key' ? 'value' : 'key');
        return;
      }
    }

    // Method selection
    if (focusArea === 'method') {
      if (key.leftArrow && methodIndex > 0) {
        setMethodIndex(methodIndex - 1);
      }
      if (key.rightArrow && methodIndex < HTTP_METHODS.length - 1) {
        setMethodIndex(methodIndex + 1);
      }
      return;
    }

    // Tag selection
    if (focusArea === 'tag' && !newTagMode) {
      if (key.leftArrow && tagIndex > 0) {
        setTagIndex(tagIndex - 1);
        setTag(availableTags[tagIndex - 1] || 'custom');
      }
      if (key.rightArrow) {
        if (tagIndex < availableTags.length - 1) {
          setTagIndex(tagIndex + 1);
          setTag(availableTags[tagIndex + 1]);
        } else if (tagIndex === availableTags.length - 1) {
          setNewTagMode(true);
          setFocusArea('newTag');
        }
      }
      return;
    }
  });

  const isTextInputFocused = () => {
    return (
      focusArea === 'name' ||
      focusArea === 'path' ||
      focusArea === 'body' ||
      focusArea === 'newTag' ||
      ((focusArea === 'queryParams' || focusArea === 'headers') && true)
    );
  };

  const moveFocusNext = () => {
    const currentIndex = focusOrder.indexOf(focusArea);
    if (currentIndex < focusOrder.length - 1) {
      const nextArea = focusOrder[currentIndex + 1];
      setFocusArea(nextArea);
      if (nextArea === 'queryParams' || nextArea === 'headers') {
        setKvFocusedIndex(0);
        setKvFocusedField('key');
      }
    }
  };

  const moveFocusPrev = () => {
    const currentIndex = focusOrder.indexOf(focusArea);
    if (currentIndex > 0) {
      const prevArea = focusOrder[currentIndex - 1];
      setFocusArea(prevArea);
      if (prevArea === 'queryParams' || prevArea === 'headers') {
        const pairs = prevArea === 'queryParams' ? queryParams : headers;
        setKvFocusedIndex(pairs.length - 1);
        setKvFocusedField('value');
      }
    }
  };

  const handleExecute = () => {
    onExecute(
      HTTP_METHODS[methodIndex],
      path,
      queryParams,
      headers,
      body || undefined
    );
  };

  const handleSave = () => {
    const finalTag = newTagMode ? newTagName : tag;
    if (!name || !finalTag) return;

    onSave({
      method: HTTP_METHODS[methodIndex],
      path,
      queryParams,
      headers,
      body,
      bodyType: 'json',
      name,
      tag: finalTag,
    });
  };

  const canSave = name && (tag || newTagName);

  return (
    <Box flexDirection="column" borderStyle="bold" borderColor="cyan" padding={1}>
      <Box paddingX={1}>
        <Text bold backgroundColor="cyan" color="black">
          {' '}MANUAL REQUEST{editingRequest ? ' (EDITING)' : ''}{' '}
        </Text>
      </Box>

      {/* Name field */}
      <Box marginTop={1} flexDirection="column">
        <Text bold color={focusArea === 'name' ? 'cyan' : undefined}>
          Name (required for saving):
        </Text>
        <Box borderStyle="single" borderColor={focusArea === 'name' ? 'cyan' : 'gray'} paddingX={1}>
          {focusArea === 'name' ? (
            <TextInput
              value={name}
              onChange={setName}
              placeholder="e.g., Get available pets"
            />
          ) : (
            <Text>{name || <Text dimColor>e.g., Get available pets</Text>}</Text>
          )}
        </Box>
      </Box>

      {/* Tag selector */}
      <Box marginTop={1} flexDirection="column">
        <Text bold color={focusArea === 'tag' || focusArea === 'newTag' ? 'cyan' : undefined}>
          Tag:
        </Text>
        <Box borderStyle="single" borderColor={focusArea === 'tag' || focusArea === 'newTag' ? 'cyan' : 'gray'} paddingX={1}>
          {newTagMode ? (
            <Box>
              <Text dimColor>New tag: </Text>
              {focusArea === 'newTag' ? (
                <TextInput
                  value={newTagName}
                  onChange={setNewTagName}
                  placeholder="Enter new tag name"
                />
              ) : (
                <Text>{newTagName || <Text dimColor>Enter new tag name</Text>}</Text>
              )}
            </Box>
          ) : (
            <Box>
              {availableTags.map((t, i) => (
                <Text key={t}>
                  {i > 0 && <Text dimColor> | </Text>}
                  <Text
                    backgroundColor={i === tagIndex && focusArea === 'tag' ? 'cyan' : undefined}
                    color={i === tagIndex && focusArea === 'tag' ? 'black' : i === tagIndex ? 'cyan' : undefined}
                  >
                    {t}
                  </Text>
                </Text>
              ))}
              <Text dimColor> | </Text>
              <Text dimColor>+ new</Text>
            </Box>
          )}
        </Box>
      </Box>

      {/* Method selector */}
      <Box marginTop={1} flexDirection="column">
        <Text bold color={focusArea === 'method' ? 'cyan' : undefined}>
          Method:
        </Text>
        <Box borderStyle="single" borderColor={focusArea === 'method' ? 'cyan' : 'gray'} paddingX={1}>
          {HTTP_METHODS.map((m, i) => (
            <Text key={m}>
              {i > 0 && <Text dimColor> | </Text>}
              <Text
                backgroundColor={i === methodIndex && focusArea === 'method' ? 'cyan' : undefined}
                color={i === methodIndex && focusArea === 'method' ? 'black' : i === methodIndex ? 'cyan' : undefined}
              >
                {m}
              </Text>
            </Text>
          ))}
        </Box>
      </Box>

      {/* URL Path */}
      <Box marginTop={1} flexDirection="column">
        <Text bold color={focusArea === 'path' ? 'cyan' : undefined}>
          URL Path:
        </Text>
        <Box borderStyle="single" borderColor={focusArea === 'path' ? 'cyan' : 'gray'} paddingX={1}>
          {focusArea === 'path' ? (
            <TextInput
              value={path}
              onChange={setPath}
              placeholder="/api/endpoint"
            />
          ) : (
            <Text>{path || <Text dimColor>/api/endpoint</Text>}</Text>
          )}
        </Box>
      </Box>

      {/* Query Parameters */}
      <Box marginTop={1}>
        <KeyValueEditor
          label="Query Parameters"
          pairs={queryParams}
          onChange={setQueryParams}
          focusedIndex={kvFocusedIndex}
          focusedField={kvFocusedField}
          isActive={focusArea === 'queryParams'}
        />
      </Box>

      {/* Headers */}
      <Box marginTop={1}>
        <KeyValueEditor
          label="Headers"
          pairs={headers}
          onChange={setHeaders}
          focusedIndex={kvFocusedIndex}
          focusedField={kvFocusedField}
          isActive={focusArea === 'headers'}
        />
      </Box>

      {/* Request Body */}
      <Box marginTop={1} flexDirection="column">
        <Text bold color={focusArea === 'body' ? 'cyan' : undefined}>
          Request Body (JSON):
        </Text>
        <Box
          borderStyle="single"
          borderColor={focusArea === 'body' ? 'cyan' : 'gray'}
          paddingX={1}
          minHeight={3}
        >
          {focusArea === 'body' ? (
            <TextInput
              value={body}
              onChange={setBody}
              placeholder='{"key": "value"}'
            />
          ) : (
            <Text>{body || <Text dimColor>{'{"key": "value"}'}</Text>}</Text>
          )}
        </Box>
      </Box>

      {/* Action buttons */}
      <Box marginTop={2} justifyContent="flex-end">
        <Text dimColor>[ Esc: Close ] </Text>
        {canSave && <Text color="cyan">[ s: Save ] </Text>}
        <Text color="green" bold>[ e: Execute ]</Text>
      </Box>
    </Box>
  );
}

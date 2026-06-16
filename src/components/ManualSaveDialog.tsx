import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';

interface ManualSaveDialogProps {
  availableTags: string[];
  initialName?: string;
  initialTag?: string;
  onSave: (name: string, tag: string) => void;
  onCancel: () => void;
}

type DialogFocus = 'name' | 'tag' | 'newTag';

export function ManualSaveDialog({ availableTags, initialName = '', initialTag, onSave, onCancel }: ManualSaveDialogProps) {
  const [name, setName] = useState(initialName);
  const [tagIndex, setTagIndex] = useState(() =>
    initialTag ? Math.max(0, availableTags.indexOf(initialTag)) : 0
  );
  const [focus, setFocus] = useState<DialogFocus>('name');
  const [newTagMode, setNewTagMode] = useState(false);
  const [newTagName, setNewTagName] = useState('');

  const currentTag = newTagMode ? newTagName : (availableTags[tagIndex] ?? '');

  useInput((input, key) => {
    if (key.escape) {
      if (newTagMode) {
        setNewTagMode(false);
        setNewTagName('');
        setFocus('tag');
        return;
      }
      onCancel();
      return;
    }

    if (focus === 'name') {
      if (key.return || key.tab) {
        setFocus('tag');
        return;
      }
    }

    if (focus === 'tag' && !newTagMode) {
      if (key.leftArrow) {
        setTagIndex(i => Math.max(0, i - 1));
        return;
      }
      if (key.rightArrow) {
        if (tagIndex < availableTags.length - 1) {
          setTagIndex(i => i + 1);
        } else {
          setNewTagMode(true);
          setFocus('newTag');
        }
        return;
      }
      if (key.tab) {
        setFocus('name');
        return;
      }
      if (key.return && name.trim() && currentTag) {
        onSave(name.trim(), currentTag);
        return;
      }
    }

    if (focus === 'newTag') {
      if (key.return && name.trim() && newTagName.trim()) {
        onSave(name.trim(), newTagName.trim());
        return;
      }
    }
  });

  return (
    <Box flexDirection="column" borderStyle="double" borderColor="cyan" paddingX={2} paddingY={1} width={60}>
      <Box justifyContent="center" marginBottom={1}>
        <Text bold color="cyan">SAVE REQUEST</Text>
      </Box>

      <Box marginBottom={1}>
        <Text dimColor>Name  </Text>
        <Box borderStyle="single" borderColor={focus === 'name' ? 'cyan' : 'gray'} flexGrow={1} paddingX={1}>
          <TextInput
            value={name}
            onChange={setName}
            focus={focus === 'name'}
            placeholder="request name"
          />
        </Box>
      </Box>

      <Box marginBottom={1}>
        <Text dimColor>Tag   </Text>
        {newTagMode ? (
          <Box borderStyle="single" borderColor="cyan" flexGrow={1} paddingX={1}>
            <TextInput
              value={newTagName}
              onChange={setNewTagName}
              focus={focus === 'newTag'}
              placeholder="new tag name"
            />
          </Box>
        ) : (
          <Box>
            <Text dimColor>← </Text>
            <Box borderStyle="single" borderColor={focus === 'tag' ? 'cyan' : 'gray'} paddingX={1} minWidth={20}>
              <Text color={currentTag ? undefined : 'red'}>{currentTag || '(no tags)'}</Text>
            </Box>
            <Text dimColor> → new</Text>
          </Box>
        )}
      </Box>

      <Box>
        <Text dimColor>Tab: switch field  ←/→: cycle tags  Enter: save  Esc: cancel</Text>
      </Box>
    </Box>
  );
}

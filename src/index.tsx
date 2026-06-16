#!/usr/bin/env node

import React from 'react';
import { withFullScreen } from 'fullscreen-ink';
import { App } from './App.js';
import { resolveCollection, listCollections, setCollectionPath } from './utils/storage.js';

const args = process.argv.slice(2);

function showHelp() {
  console.log(`
twagger - TUI Swagger/OpenAPI Documentation Viewer

Usage:
  twagger <collection>           Load from ~/.twagger/<collection>/
  twagger <spec-path-or-url>     Load from file path or URL

Examples:
  twagger PetStore
  twagger ./openapi.json
  twagger https://petstore3.swagger.io/api/v3/openapi.json

Options:
  --help, -h     Show this help message
  --version, -v  Show version number
  --list, -l     List available collections

Keyboard Shortcuts:
  Panel Navigation:
    h/Left       Focus left panel (endpoints list)
    l/Right      Focus right panel (details)

  Left Panel:
    j/k          Navigate between tags and endpoints
    Enter        Expand/collapse tag
    g            Go to first item
    G            Go to last item
    c            Collapse all tags
    x            Expand all tags

  Right Panel:
    j/k          Scroll content
    g            Scroll to top

  Actions:
    t            Toggle "Try it out" mode
    e            Execute request (in try-it-out mode)
    Esc          Cancel / go back
    m            Open manual request builder
    s            Save manual request
    Ctrl+r       Reload spec
    q            Quit
`);
}

function showVersion() {
  console.log('twagger v1.0.0');
}

async function showCollections() {
  const collections = await listCollections();
  if (collections.length === 0) {
    console.log('No collections found in ~/.twagger/');
    console.log('\nTo create a collection:');
    console.log('  mkdir -p ~/.twagger/MyAPI');
    console.log('  cp openapi.json ~/.twagger/MyAPI/');
  } else {
    console.log('Available collections:\n');
    for (const name of collections) {
      console.log(`  ${name}`);
    }
  }
}

async function main() {
  // Parse arguments
  if (args.includes('--help') || args.includes('-h')) {
    showHelp();
    process.exit(0);
  }

  if (args.includes('--version') || args.includes('-v')) {
    showVersion();
    process.exit(0);
  }

  if (args.includes('--list') || args.includes('-l')) {
    await showCollections();
    process.exit(0);
  }

  const input = args[0];

  if (!input) {
    console.error('Error: Please provide a collection name, path, or URL\n');
    showHelp();
    process.exit(1);
  }

  const collection = await resolveCollection(input);

  if (!collection) {
    console.error(`Error: Collection "${input}" not found in ~/.twagger/\n`);
    console.error('Make sure the directory exists and contains an OpenAPI spec file (JSON/YAML).\n');
    await showCollections();
    process.exit(1);
  }

  // Set collection path for overrides storage
  // For collections, path is the directory; for URLs/files, it's the source itself
  if (collection.name !== 'Remote' && collection.name !== 'Local') {
    setCollectionPath(collection.path);
  }

  const ink = withFullScreen(<App source={collection.source} collectionName={collection.name} />);
  await ink.start();
  await ink.waitUntilExit();
}

main();

#!/bin/bash
if [ ! -f "app/globals.css" ]; then
  echo "FAIL: app/globals.css does not exist"
  exit 1
fi
if ! grep -q "\-\-bg-primary" app/globals.css; then
  echo "FAIL: CSS variable --bg-primary not found in globals.css"
  exit 1
fi
if [ ! -f "next.config.mjs" ] && [ ! -f "next.config.js" ] && [ ! -f "next.config.ts" ]; then
  echo "FAIL: next config not found"
  exit 1
fi
if ! grep -q -E "output:\s*['\"]export['\"]" next.config.mjs next.config.js next.config.ts 2>/dev/null; then
  echo "FAIL: output: 'export' not found in next config"
  exit 1
fi
echo "PASS"

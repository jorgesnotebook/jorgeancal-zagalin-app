#!/bin/bash
set -e

echo "🚀 Running local CI pipeline..."
echo ""

echo "📦 Step 1: Installing dependencies..."
npm ci
echo "✅ Dependencies installed"
echo ""

echo "🔍 Step 2: Type checking..."
npm run typecheck
echo "✅ Type check passed"
echo ""

echo "🧹 Step 3: Linting..."
npm run lint
echo "✅ Lint passed"
echo ""

echo "🧪 Step 4: Unit tests..."
npm run test:ci
echo "✅ Tests passed"
echo ""

echo "🏗️  Step 5: Building frontend..."
npm run build
echo "✅ Frontend built"
echo ""

if [ -f "Magefile.go" ]; then
  echo "🔧 Step 6: Building backend..."
  if command -v mage &> /dev/null; then
    mage -v coverage
    mage -v buildAll
    echo "✅ Backend built"
  else
    echo "⚠️  Mage not installed. Skipping backend build."
    echo "   Install with: go install github.com/magefile/mage@latest"
  fi
  echo ""
fi

echo "📋 Step 7: Getting plugin metadata..."
if [ -f "dist/plugin.json" ]; then
  PLUGIN_ID=$(cat dist/plugin.json | jq -r .id)
  PLUGIN_VERSION=$(cat dist/plugin.json | jq -r .info.version)
  ARCHIVE="${PLUGIN_ID}-${PLUGIN_VERSION}.zip"
  echo "Plugin ID: $PLUGIN_ID"
  echo "Version: $PLUGIN_VERSION"
  echo "✅ Metadata retrieved"
else
  echo "❌ dist/plugin.json not found"
  exit 1
fi
echo ""

echo "📦 Step 8: Packaging plugin..."
cp -r dist "${PLUGIN_ID}"
zip -r "${ARCHIVE}" "${PLUGIN_ID}"
echo "✅ Plugin packaged: ${ARCHIVE}"
echo ""

echo "🔍 Step 9: Validating plugin metadata..."
if command -v docker &> /dev/null; then
  docker run --pull=always --rm \
    -v "$PWD/${ARCHIVE}:/archive.zip" \
    grafana/plugin-validator-cli -analyzer=metadatavalid /archive.zip
  echo "✅ Plugin metadata validated"
else
  echo "⚠️  Docker not available. Skipping metadata validation."
fi
echo ""

echo "🎉 All CI steps completed successfully!"
echo ""
echo "Next steps:"
echo "  - Review changes and commit"
echo "  - Push to trigger full CI with E2E tests"

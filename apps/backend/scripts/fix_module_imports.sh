#!/bin/bash

# Script to fix import paths in modular monolith architecture
# This updates all imports from internal/ to modules/ structure

echo "🔧 Fixing import paths in modular architecture..."

# Define the base directory
BASE_DIR="/Users/prathameshyadav/ARC-Hawk/apps/backend"

cd "$BASE_DIR" || exit 1

# Fix imports in all module files
echo "📝 Updating imports in module files..."

# Update scanning module
find modules/scanning -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/config|github.com/arc-platform/backend/internal/config|g' \
  -e 's|github.com/arc-platform/backend/internal/domain/repository|github.com/arc-platform/backend/modules/shared/interfaces|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/scanning/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/scanning/api|g' \
  {} \;

# Update assets module
find modules/assets -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/assets/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/assets/api|g' \
  {} \;

# Update lineage module
find modules/lineage -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/lineage/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/lineage/api|g' \
  {} \;

# Update compliance module
find modules/compliance -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/compliance/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/compliance/api|g' \
  {} \;

# Update masking module
find modules/masking -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/masking/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/masking/api|g' \
  {} \;

# Update analytics module
find modules/analytics -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/analytics/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/analytics/api|g' \
  {} \;

# Update connections module
find modules/connections -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure/persistence|github.com/arc-platform/backend/modules/shared/infrastructure/persistence|g' \
  -e 's|github.com/arc-platform/backend/internal/service|github.com/arc-platform/backend/modules/connections/service|g' \
  -e 's|github.com/arc-platform/backend/internal/api|github.com/arc-platform/backend/modules/connections/api|g' \
  {} \;

# Update shared module
find modules/shared -name "*.go" -type f -exec sed -i '' \
  -e 's|github.com/arc-platform/backend/internal/domain/entity|github.com/arc-platform/backend/modules/shared/domain/entity|g' \
  -e 's|github.com/arc-platform/backend/internal/infrastructure|github.com/arc-platform/backend/modules/shared/infrastructure|g' \
  -e 's|github.com/arc-platform/backend/internal/domain/repository|github.com/arc-platform/backend/modules/shared/interfaces|g' \
  {} \;

echo "✅ Import paths updated successfully!"

# Verify no old imports remain
echo "🔍 Checking for remaining old imports..."
OLD_IMPORTS=$(grep -r "github.com/arc-platform/backend/internal" modules/ | grep -v "config" | wc -l)

if [ "$OLD_IMPORTS" -gt 0 ]; then
    echo "⚠️  Warning: Found $OLD_IMPORTS old import references (excluding config)"
    grep -r "github.com/arc-platform/backend/internal" modules/ | grep -v "config" | head -10
else
    echo "✅ No old imports found!"
fi

echo "🎉 Import path fix complete!"

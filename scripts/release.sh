#!/bin/bash
# Release script - creates a new semver tag

set -e

# Get current version or default to 0.0.0
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
CURRENT_VERSION=${CURRENT_VERSION#v}  # Remove 'v' prefix

echo "Current version: $CURRENT_VERSION"
echo ""
echo "Bump type:"
echo "  1) Patch (bug fixes)         $CURRENT_VERSION -> $(echo $CURRENT_VERSION | awk -F. '{print $1"."$2"."$3+1}')"
echo "  2) Minor (new features)      $CURRENT_VERSION -> $(echo $CURRENT_VERSION | awk -F. '{print $1"."$2+1".0"}')"
echo "  3) Major (breaking changes)  $CURRENT_VERSION -> $(echo $CURRENT_VERSION | awk -F. '{print $1+1".0.0"}')"
echo "  4) Custom version"
echo ""
read -p "Select (1-4): " choice

case $choice in
  1)
    NEW_VERSION=$(echo $CURRENT_VERSION | awk -F. '{print $1"."$2"."$3+1}')
    ;;
  2)
    NEW_VERSION=$(echo $CURRENT_VERSION | awk -F. '{print $1"."$2+1".0"}')
    ;;
  3)
    NEW_VERSION=$(echo $CURRENT_VERSION | awk -F. '{print $1+1".0.0"}')
    ;;
  4)
    read -p "Enter version (without 'v' prefix): " NEW_VERSION
    ;;
  *)
    echo "Invalid choice"
    exit 1
    ;;
esac

# Validate semver format
if ! [[ $NEW_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$ ]]; then
  echo "Error: Invalid semver format: $NEW_VERSION"
  exit 1
fi

TAG="v${NEW_VERSION}"

echo ""
echo "Creating tag: $TAG"
read -p "Enter release notes (optional): " NOTES

if [ -z "$NOTES" ]; then
  NOTES="Release $TAG"
fi

# Create annotated tag
git tag -a "$TAG" -m "$NOTES"

echo ""
echo "Tag created: $TAG"
echo ""
echo "Next steps:"
echo "  1. Review: git show $TAG"
echo "  2. Push:   git push origin $TAG"
echo ""
echo "After pushing the tag, GitHub Actions will automatically:"
echo "  - Run tests"
echo "  - Build binaries for Linux, macOS, and Windows"
echo "  - Create a GitHub release with binaries attached"
echo "  - Generate release notes from commits"

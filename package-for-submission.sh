#!/bin/bash

# Package Rate Limiter Service for Submission
# This script creates a ZIP file with all project files

echo "📦 Packaging Rate Limiter Service for submission..."

# Set the output filename
OUTPUT_FILE="rate-limiter-service-submission.zip"

# Remove old ZIP if it exists
if [ -f "$OUTPUT_FILE" ]; then
    echo "🗑️  Removing old submission file..."
    rm "$OUTPUT_FILE"
fi

# Create ZIP file excluding unnecessary files
echo "🗜️  Creating ZIP archive..."
zip -r "$OUTPUT_FILE" . \
    -x "*.git*" \
    -x "*node_modules*" \
    -x "*build/*" \
    -x "*.dart_tool/*" \
    -x "*flutter_dashboard/.dart_tool/*" \
    -x "*flutter_dashboard/build/*" \
    -x "*flutter_dashboard/.flutter-plugins" \
    -x "*flutter_dashboard/.flutter-plugins-dependencies" \
    -x "*flutter_dashboard/.metadata" \
    -x "*.DS_Store" \
    -x "*__pycache__*" \
    -x "*.pyc" \
    -x "*vendor/*" \
    -x "*coverage/*" \
    -x "*.env" \
    -x "*tmp/*" \
    -x "*temp/*" \
    -x "*.log" \
    -x "$OUTPUT_FILE"

# Check if ZIP was created successfully
if [ -f "$OUTPUT_FILE" ]; then
    echo "✅ Package created successfully: $OUTPUT_FILE"
    echo ""
    echo "📊 Package details:"
    ls -lh "$OUTPUT_FILE"
    echo ""
    echo "📋 Contents preview:"
    unzip -l "$OUTPUT_FILE" | head -30
    echo ""
    echo "✨ Ready for submission!"
else
    echo "❌ Error: Failed to create package"
    exit 1
fi

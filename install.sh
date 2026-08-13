#!/bin/bash
set -e

# ==============================================================================
# SpeedMap Build & Install to /Applications
# Allows launching directly from macOS Applications / Dock bar / Spotlight
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

APP_NAME="SpeedMap.app"
BUILD_OUTPUT="$SCRIPT_DIR/build/bin/$APP_NAME"
TARGET_APP="/Applications/$APP_NAME"

echo "⚡ [1/4] Збирання додатку SpeedMap..."
wails build

if [ ! -d "$BUILD_OUTPUT" ]; then
    echo "❌ Помилка: $BUILD_OUTPUT не знайдено після білду."
    exit 1
fi

echo "🔄 [2/4] Зупинка попередньої версії SpeedMap (якщо запущена)..."
pkill -x SpeedMap 2>/dev/null || true
sleep 1

echo "📦 [3/4] Переміщення $APP_NAME у /Applications..."
rm -rf "$TARGET_APP"
cp -R "$BUILD_OUTPUT" "/Applications/"
xattr -cr "$TARGET_APP" 2>/dev/null || true

echo "🚀 [4/4] Запуск SpeedMap з /Applications..."
open "$TARGET_APP"

echo ""
echo "✅ Готово! Додаток успішно встановлено в /Applications/SpeedMap.app"
echo "📌 Тепер ви можете закріпити SpeedMap у Dock-барі або відкривати через Spotlight (Cmd + Space)."

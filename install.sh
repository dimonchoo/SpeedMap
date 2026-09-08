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

echo "🧩 [1/5] Компіляція та валідація модульного HTML інтерфейсу..."
node "$SCRIPT_DIR/scripts/build-html.js"

echo "⚡ [2/5] Збирання додатку SpeedMap..."
wails build

if [ ! -d "$BUILD_OUTPUT" ]; then
    echo "❌ Помилка: $BUILD_OUTPUT не знайдено після білду."
    exit 1
fi

echo "🔄 [3/5] Зупинка попередньої версії SpeedMap (якщо запущена)..."
pkill -x SpeedMap 2>/dev/null || true
sleep 1

echo "📦 [4/5] Переміщення $APP_NAME у /Applications..."
rm -rf "$TARGET_APP"
cp -R "$BUILD_OUTPUT" "/Applications/"
xattr -cr "$TARGET_APP" 2>/dev/null || true

echo "🚀 [5/5] Запуск SpeedMap з /Applications..."
open "$TARGET_APP"

echo ""
echo "✅ Готово! Додаток успішно встановлено в /Applications/SpeedMap.app"
echo "📌 Тепер ви можете закріпити SpeedMap у Dock-барі або відкривати через Spotlight (Cmd + Space)."

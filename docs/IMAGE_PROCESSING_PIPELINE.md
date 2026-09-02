# 🖼️ Повний технічний регламент та специфікація пайплайну обробки зображень у SpeedMap

Цей документ містить вичерпний, покроковий опис усіх правил, математичних порогів, евристик, регулярних виразів та умов, за яких зображення виявляються, фільтруються, декодуються, ресайзяться, оптимізуються, упаковуються та перевіряються на регресію.

---

## 📑 Зміст
1. [Етап 1: Виявлення зображень у браузері (Discovery & DOM Crawler)](#етап-1-виявлення-зображень-у-браузері-discovery--dom-crawler)
2. [Етап 2: Фільтрація трекерів, маяків та виключень (Beacon & Pattern Filtering)](#етап-2-фільтрація-трекерів-маяків-та-виключень-beacon--pattern-filtering)
3. [Етап 3: Крос-сторінкова агрегація та метрики (Cross-Page Aggregation)](#етап-3-крос-сторінкова-агрегація-та-метрики-cross-page-aggregation)
4. [Етап 4: Відбір для експорту та дедуплікація (Export Filter & Deduplication)](#етап-4-відбір-для-експорту-та-дедуплікація-export-filter--deduplication)
5. [Етап 5: Завантаження та виправлення альфа-каналу (Download & Straight RGBA)](#етап-5-завантаження-та-виправлення-альфа-каналу-download--straight-rgba)
6. [Етап 6: Ресайз під Retina 2x (Retina Downscaling)](#етап-6-ресайз-під-retina-2x-retina-downscaling)
7. [Етап 7: Адаптивний Decision Engine (Кодування WebP, Lossless vs Lossy, Fallback)](#етап-7-адаптивний-decision-engine-кодування-webp-lossless-vs-lossy-fallback)
8. [Етап 8: Генерація WordPress Deploy Package та артефактів](#етап-8-генерація-wordpress-deploy-package-та-артефактів)
9. [Етап 9: Трекер регресій та зіставлення маніфестів (Regression & Diff Engine)](#етап-9-трекер-регресій-та-зіставлення-маніфестів-regression--diff-engine)

---

## Етап 1: Виявлення зображень у браузері (Discovery & DOM Crawler)

Краулер на базі Chrome DevTools Protocol інспектує кожну сторінку у двох паралельних шарах:

### 1.1. Шар 1: Network & PerformanceObserver
Перехоплює всі мережеві ресурси `performance.getEntriesByType('resource')`:
* **Умова визначення типу**:
  * `r.initiatorType === 'img' || r.initiatorType === 'image' || r.initiatorType === 'css' || r.initiatorType === 'picture'`
  * **АБО** розширення URL відповідає регулярному виразу:
    ```regex
    \.(jpg|jpeg|png|webp|avif|gif|svg|ico|bmp)(\?.*)?$
    ```
* **Параметри, що вилучаються**:
  * `transferSize = r.transferSize || r.encodedBodySize || r.decodedBodySize || 0`
  * `encodedSize = r.encodedBodySize || 0`
  * `duration = Math.round(r.duration || 0)` (у мілісекундах)

### 1.2. Шар 2: Глибинний DOM Inspector
Знаходить усі візуальні та приховані зображення, які ще не завантажилися через мережу (lazy-load):
1. **Звичайні теги `<img>`**:
   * Джерела: `img.currentSrc || img.src || img.getAttribute('data-src')`
   * Геометрія: `img.naturalWidth`, `img.naturalHeight`, та `getBoundingClientRect()` (`renderedWidth`, `renderedHeight`).
   * Прапорець lazy-loading:
     ```javascript
     isLazy = img.getAttribute('loading') === 'lazy' || !!img.getAttribute('data-src')
     ```
2. **Адаптивні теги `<picture>` та атрибути `srcset`**:
   * Селектор: `picture source, img[srcset], img[data-srcset], [data-lazy-src], [data-original], [data-hi-res], [data-full-url]`
   * Парсить усі запчастини `srcset`, розбиваючи за комами та пробілами `srcset.split(',')[i].trim().split(/\s+/)[0]`.
3. **Фонові зображення CSS (Background Images)**:
   * Селектор: `[style*="url"], [data-bg], [data-background], [data-bg-url], style`
   * Регулярний вираз вилучення URL з CSS:
     ```regex
     url\(\s*['"]?([^'")]+?\.(?:png|jpg|jpeg|webp|avif|gif|bmp))['"]?\s*\)
     ```
4. **Постери відео та мета-теги**:
   * Селектор: `video[poster], link[rel="preload"][as="image"], meta[property="og:image"], meta[name="twitter:image"], [data-image], [data-img], [data-thumb], [data-slide], [data-slide-bg]`

---

## Етап 2: Фільтрація трекерів, маяків та виключень (Beacon & Pattern Filtering)

Щоб уникнути потрапляння аналітичних пікселів 1x1, рекламних скриптів та сторонніх віджетів у пайплайн оптимізації:

### 2.1. Список вбудованих заборонених патернів (якщо `FilterTrackingBeacons == true`):
* `googleadservices.com`
* `doubleclick.net`
* `facebook.com/tr`
* `bat.bing.com`
* `clarity.ms`
* `px.ads.linkedin.com`
* `analytics.google.com`
* `google-analytics.com`
* `t.co/1/i/adsct`
* `stats.wp.com`
* `/pagead/`
* Плюс будь-які користувацькі рядки з налаштування `ExcludedImagePatterns`.

### 2.2. Умова відсікання:
```javascript
if (url.toLowerCase().includes(pattern.toLowerCase().trim())) {
    // Ресурс ігнорується, не потрапляє у діагностику сторінки
}
```

---

## Етап 3: Крос-сторінкова агрегація та метрики (Cross-Page Aggregation)

Після сканування всіх сторінок сайту `ComputeSiteAnalytics` формує єдиний зведений каталог зображень `SiteAnalytics.AllImages`:

### 3.1. Об'єднання дублікатів по URL:
* Для кожного унікального `img.URL`:
  * `MaxTransferSize = max(transferSize across all pages)`
  * `Pages = unique list of all pages containing this image`
  * `PageCount = len(Pages)`
  * `MaxRenderedWidth = max(renderedWidth across all pages)`
  * `MaxRenderedHeight = max(renderedHeight across all pages)`
  * `NaturalWidth = max(naturalWidth)`
  * `NaturalHeight = max(naturalHeight)`

### 3.2. Розрахунок рекомендованих Retina-розмірів:
* **Формула**:
  $$\text{RetinaW} = \text{MaxRenderedWidth} \times 2$$
  $$\text{RetinaH} = \text{MaxRenderedHeight} \times 2$$
* **Правило безпеки (Strict Ceiling)**: Зображення **ніколи не збільшується (upscale)** штучно:
  $$\text{RecommendedRetinaWidth} = \min(\text{RetinaW}, \text{NaturalWidth})$$
  $$\text{RecommendedRetinaHeight} = \min(\text{RetinaH}, \text{NaturalHeight})$$

### 3.3. Визначення `IsOversized`:
$$\text{IsOversized} = \text{true} \iff \text{NaturalWidth} > (\text{MaxRenderedWidth} \times 2)$$

### 3.4. Визначення `IsHeavy`:
$$\text{IsHeavy} = \text{true} \iff \text{MaxTransferSize} \ge \text{thresholdBytes}$$
* За замовчуванням: `HeavyImageThresholdKB = 100` $\implies \text{thresholdBytes} = 102,400\text{ байт}$.

---

## Етап 4: Відбір для експорту та дедуплікація (Export Filter & Deduplication)

Функція `wpexport.CollectHeavyImages` формує фінальний список кандидатів на оптимізацію за такими правилами:

### 4.1. Форматний фільтр:
* Дозволені растрові формати:
  ```go
  var rasterFormats = map[string]bool{
      "png": true, "jpg": true, "jpeg": true, "gif": true, "bmp": true,
  }
  ```
* Заборонені (пропускаються): `svg`, `webp`, `avif`.

### 4.2. Відновлення майстер-оригіналу (`preferOriginalURL`):
WordPress генерує нарізки розмірів (наприклад, `hero-1024x768.png` або `banner-300x250.jpg`).
* **Регулярний вираз суфікса розміру**:
  ```regex
  -\d+x\d+(\.[A-Za-z0-9]+)(\?.*)?$
  ```
* Якщо знайдено збіг, суфікс відрізається: `hero-1024x768.png` $\rightarrow$ `hero.png`.
* Це гарантує, що оптимізується саме вихідний файл максимальної чіткості, а не розмита зменшена копія.

### 4.3. Розрахунок відносного шляху у WordPress (`webpRel`):
* Якщо в URL є маркер `/wp-content/uploads/`:
  * Вилучається підкаталог: `/wp-content/uploads/2026/02/banner.png` $\rightarrow$ `2026/02/banner.webp`.
* Якщо маркер відсутній:
  * Використовується чисте базове ім'я: `banner.webp`.

### 4.4. Дедуплікація однойменних файлів (PNG vs JPG collision):
* Якщо для одного й того ж ключа `strings.ToLower(webpRel)` на сайті є кілька файлів (наприклад, `hero.png` та `hero.jpg`):
  * Обирається файл із **найбільшим розміром (`MaxTransferSize`)**.
  * Якщо один файл має суфікс розміру, а інший — ні, обирається файл **без суфікса**.

### 4.5. Критерій включення у список важких:
$$\text{Включити} \iff (\text{im.IsHeavy} == \text{true}) \lor (\text{im.Bytes} == 0)$$
*(Включаються підтверджені важкі файли $\ge 100\text{ KB}$ або lazy-load зображення, чий розмір не вдалося визначити в браузері).*

---

## Етап 5: Завантаження та виправлення альфа-каналу (Download & Straight RGBA)

Під час завантаження кожного зображення:

### 5.1. Повторна верифікація реального розміру на диску:
* Якщо реальний розмір завантаженого тіла HTTP `< 102,400 байт` (`HeavyImageThresholdKB`):
  * Файл **пропускається**:
    ```text
    [wpexport] skip <URL>: downloaded size X < threshold 102400 B
    ```

### 5.2. Усунення бага чорних контурів Go (`toStraightRGBA`):
* **Проблема**: Стандартний декодер Go (`image.Decode`) при читанні PNG із прозорістю автоматично множить кольори на альфа-канал (**premultiplied alpha**). При передачі в `libwebp` C-API напівпрозорі антиаліасинг-пікселі (тіні, заокруглені кути кнопок) перетворювалися на брудні чорні обідки.
* **Рішення**: Функція `toStraightRGBA`:
  * Для `image.NRGBA`: копіює байти `Pix` напряму в `image.RGBA` без премультиплікації:
    ```go
    rgba := &image.RGBA{
        Pix:    make([]uint8, len(nrgba.Pix)),
        Stride: nrgba.Stride,
        Rect:   nrgba.Rect,
    }
    copy(rgba.Pix, nrgba.Pix)
    return rgba
    ```
  * Для інших моделей: попіксельно вилучає прямі значення `color.NRGBA{R, G, B, A}`.

---

## Етап 6: Ресайз під Retina 2x (Retina Downscaling)

Якщо увімкнено `ResizeToRetina` (за замовчуванням `true`) і передано `maxW > 0` або `maxH > 0`:

### 6.1. Умова спрацьовування:
$$\text{Ресайзити} \iff (\text{origW} > \text{maxW}) \lor (\text{origH} > \text{maxH})$$
*(Якщо оригінал вже менший за ліміти, ресайз пропускається).*

### 6.2. Розрахунок коефіцієнта масштабування (Збереження Aspect Ratio):
$$\text{scaleW} = \frac{\text{maxW}}{\text{origW}}, \quad \text{scaleH} = \frac{\text{maxH}}{\text{origH}}$$
$$\text{scale} = \min(\text{scaleW}, \text{scaleH})$$
$$\text{targetW} = \max(1, \operatorname{round}(\text{origW} \times \text{scale}))$$
$$\text{targetH} = \max(1, \operatorname{round}(\text{origH} \times \text{scale}))$$

### 6.3. Інтерполяція:
Використовується високоякісний бікубічний фільтр **Catmull-Rom**:
```go
draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
```

---

## Етап 7: Адаптивний Decision Engine (Кодування WebP, Lossless vs Lossy, Fallback)

Це центральне ядро оптимізатора (`ConvertImageToWebPWithBudget`). Воно працює за суворою каскадною логікою:

```mermaid
flowchart TD
    Start["Початок обробки"] --> BudgetCalc["Розрахунок safeBudget = 85% від 100 KB = 85 KB"]
    BudgetCalc --> LossyBase["1. Базове кодування Lossy WebP (Quality = 85%)"]
    
    LossyBase --> AscendCheck{"LossyLen < 70% safeBudget (59.5 KB)<br/>ТА OrigSize > 80 KB<br/>ТА Quality < 96%?"}
    AscendCheck -->|Так| AscendLoop["Quality Ascension Loop:<br/>Тест Q = [88, 91, 93, 95, 96]%"]
    AscendLoop --> AscendCond{"highQLen ≤ 85 KB<br/>ТА highQLen < OrigSize * 60%?"}
    AscendCond -->|Так| CommitHighQ["Застосувати вищу якість WebP"]
    AscendCond -->|Ні / Бюджет перевищено| BreakAscend["Зупинити підйом якості"]
    AscendCheck -->|Ні| FormatBranch
    CommitHighQ --> AscendLoop
    BreakAscend --> FormatBranch

    FormatBranch{"2. PNG або є прозорість?<br/>ТА adaptive == true"}
    FormatBranch -->|Так| EncodeLossless["Тестове кодування Lossless WebP (Exact: true, Q=100)"]
    FormatBranch -->|Ні (JPG)| FallbackCheck
    
    EncodeLossless --> LosslessCond{"LosslessLen < OrigSize<br/>ТА LosslessLen ≤ 85 KB (safeBudget)?"}
    LosslessCond -->|Так (Іконки, Лого, UI)| SelectLossless["Обрати Lossless WebP (Q=100)"]
    LosslessCond -->|Ні (Фото-PNG, великі банери)| KeepLossy["Залишити Lossy WebP (20-60 KB)"]

    SelectLossless --> FallbackCheck
    KeepLossy --> FallbackCheck

    FallbackCheck{"3. WebP Size ≥ OrigSize?"}
    FallbackCheck -->|Так (Регресія ваги)| StepDownLoop["Step-Down Fallback Loop:<br/>Тест Q = [90, 85, 80, 75, 70, 65, 60]%"]
    StepDownLoop --> StepDownSuccess{"fBytes < OrigSize?"}
    StepDownSuccess -->|Так| CommitStepDown["Зберегти знижену якість (WebP < Orig)"]
    StepDownSuccess -->|Ні| NextQ["Спробувати нижчий Q"]
    StepDownLoop -->|Вичерпано / WebP ≥ Orig| SkipCheck
    CommitStepDown --> Done

    FallbackCheck -->|Ні| Done["Фінальний WebP готовий"]

    SkipCheck{"skipIfNoSavings == true?"}
    SkipCheck -->|Так| MarkSkipped["Позначити IsSkipped = true, Savings = 0"]
    SkipCheck -->|Ні| Done
    MarkSkipped --> Done
```

### 7.1. Розрахунок безпечного бюджету (`safeBudget`):
$$\text{safeBudget} = \max(40\text{ KB}, \operatorname{round}(\text{budgetBytes} \times 0.85))$$
*(При бюджеті 100 КБ $\implies \text{safeBudget} = 85\text{ KB} = 87,040\text{ байт}$).*

### 7.2. Базове кодування Lossy WebP:
* Початкова якість: `Quality = 85.0%`.
* Опції: `&webp.Options{ Lossless: false, Quality: 85, Exact: false }`.

### 7.3. Підйом якості (Target-Budget Optimization):
* **Умова активації**:
  $$\text{adaptive} \land (\text{origSize} > 80\text{ KB}) \land (\text{lossyLen} < \text{safeBudget} \times 0.70) \land (\text{quality} < 96)$$
* **Рівні тестування якості**: `[88.0, 91.0, 93.0, 95.0, 96.0]`.
* **Умова фіксації вищої якості**:
  $$\text{highQLen} \le \text{safeBudget} \quad \land \quad \text{highQLen} < (\text{origSize} \times 0.60)$$
  *(Тобто файл залишається $\le 85\text{ KB}$ та забезпечує $\ge 40\%$ чистої економії).*
* Як тільки розмір перевищує ліміт — цикл миттєво переривається (`break`).

### 7.4. Адаптивна маршрутизація Lossless vs Lossy:
* **Умова тестування**: `(formatName == "png" || isTransparent) && adaptive`.
* Кодується тестовий буфер: `&webp.Options{ Lossless: true, Exact: true }`.
* **Математичне правило вибору Lossless**:
  $$\text{Обрати Lossless} \iff (\text{losslessLen} < \text{origSize}) \land (\text{losslessLen} \le \text{safeBudget})$$
* **Результат**:
  * Дрібні векторні іконки, логотипи, кнопки та бейджі ($\le 85\text{ KB}$) зберігаються в **100% Lossless** без жодних артефактів.
  * Величезні 5-мегапіксельні фото-PNG (`Voice-of-the-Buyer`, `Depositphotos`), чий Lossless розмір сягає 1.5–2.5 МБ, **примусово залишаються у Lossy WebP** (стискаючись до 20–60 КБ із -98% економії).

### 7.5. Захист від роздування розміру (Step-Down Fallback):
* **Умова активації**: `if webpSize >= origSize` (наприклад, складний ілюстрований слайдер на кшталт `Case-Study_Slider`).
* **Каскад зниження якості**:
  ```go
  fallbackQualities := []float32{90.0, 85.0, 80.0, 75.0, 70.0, 65.0, 60.0}
  ```
* Як тільки на будь-якому кроці $\text{len}(\text{fBytes}) < \text{origSize}$, цей рівень фіксується і цикл зупиняється.

### 7.6. Пропуск файлів без виграшу (`skipIfNoSavings`):
* Якщо навіть на мінімальній допустимій планці якості (`minQuality = 75%` або `60%`) розмір WebP не став меншим за оригінал:
  * Якщо `skipIfNoSavings == true`:
    $$\text{IsSkipped} = \text{true}, \quad \text{SavingsBytes} = 0, \quad \text{SavingsPercent} = 0$$
  * Файл не замінюється на сервері, запобігаючи деградації ваги сайту.

---

## Етап 8: Генерація WordPress Deploy Package та артефактів

Функція `WriteDeployPackage` створює автономну структуру папки `speedmap-webp-YYYYMMDD-HHMMSS`:

```
speedmap-webp-20260902-171531/
├── images/
│   ├── 001/optimized.webp
│   ├── 002/optimized.webp
│   └── ... (лише успішно оптимізовані файли)
├── manifest.json
├── apply.php
├── rollback.php
├── compare.html
└── render-report.html
```

### 8.1. Структура `manifest.json`:
Для кожного файлу фіксується повний контекст:
```json
{
  "sourceUrl": "https://infuse.com/wp-content/uploads/2026/02/hero.png",
  "webpRel": "2026/02/hero.webp",
  "wpUploadRel": "2026/02/hero.webp",
  "originalBytes": 316416,
  "optimizedBytes": 223027,
  "savingsPercent": 29.5,
  "naturalWidth": 1770,
  "naturalHeight": 700,
  "recommendedRetinaWidth": 1770,
  "recommendedRetinaHeight": 700,
  "isLossless": false,
  "quality": 85.0,
  "pages": [
    "https://infuse.com/insights/",
    "https://infuse.com/case-studies/"
  ]
}
```

### 8.2. Правило оптимізації ваги архіву (Вилучення дублікатів оригіналів):
* **Проблема минулих версій**: Архіви важили по 390–500 МБ, бо дублювали всі вихідні PNG/JPG у папці `images/*/original.*`.
* **Рішення**: Оригінали **не записуються в пакет**. В `compare.html` вони завантажуються онлайн за URL напряму з продакшну.
* **Результат**: Вага deploy-пакету скоротилася з **390 МБ до ~46 МБ (-87%)**.

---

## Етап 9: Трекер регресій та зіставлення маніфестів (Regression & Diff Engine)

Кожен згенерований пакет автоматично реєструється в глобальному реєстрі `~/.speedmap/exports.json`:

```json
{
  "id": "speedmap-webp-20260902-171531",
  "domain": "https://infuse.com/sitemap.xml",
  "timestamp": "2026-09-02T14:15:33Z",
  "formattedTime": "02.09.2026 17:15:33",
  "packageDir": "/Users/.../speedmap-webp-20260902-171531",
  "manifestPath": "/Users/.../speedmap-webp-20260902-171531/manifest.json",
  "imageCount": 712,
  "originalBytes": 344131580,
  "optimizedBytes": 48863616,
  "savingsPercent": 85.8,
  "existsOnDisk": true
}
```

### 9.1. Математичні пороги класифікації регресій (`CompareExportPackages`):
Для кожного спільного файлу між Базовим (Base) та Поточним (Current) маніфестом обчислюється різниця:
$$\Delta = \text{Current.OptimizedBytes} - \text{Base.OptimizedBytes}$$

| Статус | Умова | Колір бейджа в UI | Опис |
| :--- | :--- | :--- | :--- |
| 🔴 **`degraded`** | $\Delta > 5 \times 1024\text{ байт}$ ($+5\text{ KB}$) | Червоний | Регресія розміру (файл поважчав) |
| 🟢 **`improved`** | $\Delta < -5 \times 1024\text{ байт}$ ($-5\text{ KB}$) | Зелений | Покращення (файл став легшим) |
| ⚪ **`same`** | $-5\text{ KB} \le \Delta \le +5\text{ KB}$ | Сірий | Розмір практично не змінився |
| 🆕 **`new`** | Файл є в поточному, але відсутній у базовому | Синій | Нове зображення |
| 🗑️ **`removed`** | Файл був у базовому, але відсутній у поточному | Жовтий | Видалене або відфільтроване |

### 9.2. Сортування результатів:
Усі погіршені файли (`degraded`) автоматично сортуються **в самий верх списку** за спаданням величини регресії ($\Delta$), що дозволяє команді миттєво побачити причину будь-якого збільшення розміру пакету.

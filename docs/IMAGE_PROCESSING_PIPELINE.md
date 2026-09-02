# 🖼️ SpeedMap Image Processing & Optimization Pipeline Architecture

This document provides a comprehensive, end-to-end technical breakdown of how **SpeedMap** discovers, aggregates, resizes, optimizes, encodes, packages, and tracks regression metrics for website images.

---

```mermaid
flowchart TD
    A["🕷️ 1. Discovery (Crawler)"] -->|Collect DOM & Network Images| B["📊 2. Aggregation & Metrics"]
    B -->|Track Max Rendered Dimensions & Pages| C["🎯 3. Heavy Image Filtering (≥100 KB)"]
    C -->|Deduplicate by WebP target path| D["📥 4. Parallel Download & RGBA Normalization"]
    D -->|toStraightRGBA (Fixes Dark Border Bug)| E["📐 5. Retina 2x Proportional Resizing"]
    E -->|Constrain to 2x Max Rendered Dimensions| F["🧠 6. Adaptive Encoding Engine"]
    
    subgraph "Adaptive Decision Engine"
        F --> G["Encode Base Lossy WebP (Quality 75-85)"]
        G --> H{"Target Budget Test (<70% of 100KB)?"}
        H -->|Yes| I["Quality Ascend (Test 88-96% Q)"]
        H -->|No / Done| J{"PNG or Transparent Asset?"}
        
        J -->|Yes| K["Encode Lossless WebP"]
        K --> L{"Lossless ≤ 85 KB & < Original?"}
        L -->|Yes (Icons/Badges)| M["Select Lossless WebP (Q=100)"]
        L -->|No (Photos/Banners)| N["Keep Lossy WebP"]
        J -->|No (JPG)| N
        
        M --> O{"WebP Size ≥ Original?"}
        N --> O
        O -->|Yes| P["Step-Down Fallback (Down to 60% Q)"]
        O -->|No| Q["Optimal WebP Selected"]
        P --> Q
    end

    Q --> R["📦 7. WordPress Deploy Package"]
    R --> S["manifest.json + apply.php + rollback.php + compare.html"]
    R --> T["~/.speedmap/exports.json (Regression Registry)"]
```

---

## 1. Discovery & Multi-Page Observation

During the sitemap crawl, headless Chrome navigates to each page and gathers images from two distinct layers:
1. **DOM Inspection**: Every `<img>` tag, `<picture>` source, inline SVG, background image in CSS, and `srcset` candidate.
   * Extracts: `naturalWidth`, `naturalHeight`, `src`, `loading="lazy"`, `alt`, and `getBoundingClientRect()` (`renderedWidth`, `renderedHeight`).
2. **PerformanceObserver Network Intercept**: Intercepts exact transfer sizes (`transferSize`), encoded resource sizes (`encodedSize`), and network response times (`duration`).

---

## 2. Cross-Page Aggregation & Dimension Tracking

Because the same image (e.g., header logo, author portrait, global banner) often appears on dozens of different pages with varying viewport layouts:
* **Max Rendered Bounds (`maxRenderedWidth`, `maxRenderedHeight`)**: SpeedMap tracks the **maximum** DOM display size the image occupies across *all* crawled pages.
* **Page List (`pages`)**: Gathers all URLs referencing the image for contextual review.
* **Format Detection**: Normalizes format (`png`, `jpg`, `webp`, `svg`, `avif`, `gif`).

---

## 3. Filtering & Deduplication

SpeedMap applies a strict filtering stage to avoid wasting compute on trivial or non-convertible assets:
* **Format Gate**: Only raster formats (`png`, `jpg`, `jpeg`, `gif`, `bmp`) are converted. Vectors (`svg`) and pre-optimized next-gen formats (`webp`, `avif`) are preserved.
* **Threshold Gate (`HeavyImageThresholdKB: 100 KB`)**:
  * Default: **100 KB** (`102,400 bytes`).
  * Only images whose true on-disk / network size exceeds this threshold are packaged for WordPress deployment.
* **Stem Deduplication (`webpRel`)**:
  * If both `hero.png` and `hero.jpg` map to `2026/08/hero.webp`, SpeedMap deduplicates them to a single target WebP entry, prioritizing the highest-resolution original.
  * Resized WordPress thumbnails (`hero-1024x768.png`) are mapped back to their master original (`hero.png`).

---

## 4. Download & Straight RGBA Normalization

When downloading images for processing:
* **Authentication**: Basic Auth credentials (`AuthUser`, `AuthPass`) and custom HTTP headers are injected if configured.
* **Go `toStraightRGBA` Color Fix**:
  > **Note**: Standard Go `image.Decode` converts images into **premultiplied alpha** (`image.NRGBA` → `image.RGBA`). When passed to Google's `libwebp` C-API, premultiplied edge pixels cause dark/black borders around anti-aliased semi-transparent graphics (drop shadows, rounded icons, transparent logos).
  > 
  > SpeedMap passes all decoded pixels through `toStraightRGBA()` to preserve raw, unpremultiplied straight RGBA color values, ensuring pixel-perfect alpha blending.

---

## 5. Retina 2x Proportional Resizing

To fulfill Google Lighthouse's **"Properly Size Images"** Core Web Vital metric:
* If an image is 4000x2500 px, but across the entire website it is never displayed larger than 600x375 px:
  * SpeedMap calculates the Retina target dimensions: 600 × 2 = 1200 px.
  * The image is downscaled proportionally using high-quality Catmull-Rom interpolation (`golang.org/x/image/draw.BiLinear / CatmullRom`).
* **Aspect Ratio Preservation**: Natural aspect ratio is strictly preserved; images are never distorted or stretched.

---

## 6. The Adaptive Decision & Encoding Engine

The core optimization engine balances **zero visual artifacts** with **extreme file size reduction**:

```
                                  Input Image
                                       │
                         ┌─────────────┴─────────────┐
                         ▼                           ▼
                 Lossy WebP Engine           Lossless WebP Engine
                 (Quality 75 - 85)             (Exact RGB 4:4:4)
                         │                           │
                         ▼                           ▼
                 [ lossyData ]               [ losslessData ]
                         │                           │
                         └─────────────┬─────────────┘
                                       │
                      ┌────────────────┴────────────────┐
                      ▼                                 ▼
           Is it a Small UI Asset?           Is it a Photo / Slide?
           (lossless ≤ 85 KB)                (lossless > 85 KB)
                      │                                 │
                      ▼                                 ▼
               Use LOSSLESS WebP                 Use LOSSY WebP
            (Crisp text & sharp edges)       (20-60 KB, -90% savings)
```

### 1. Base Lossy WebP Encoding
* Encodes with base quality (default **85%**, minimum floor **75%**).

### 2. Target-Budget Optimization (Quality Ascension)
* If an image compresses to a tiny file size well below the budget (e.g., < 70 KB), SpeedMap tests higher quality levels (**88%, 91%, 93%, 95%, 96%**).
* If the file remains within budget and maintains ≥ 40% savings, the higher-quality WebP is selected, eliminating compression noise and banding.

### 3. Adaptive Lossless Routing
* **Lossless WebP (`Q=100, exact=true`)** maintains uncompressed RGB 4:4:4 color fidelity without chroma subsampling.
* **Routing Rule**: Lossless is chosen **if and only if** `losslessBytes <= safeBudget (85 KB)` and `losslessBytes < originalBytes`.
  * **Result**: Small badges, UI icons, navigation buttons, and transparent logos get 100% Lossless crispness.
  * **Result**: 5-megapixel photographic PNGs (`Voice-of-the-Buyer`, `Depositphotos`) are routed to Lossy WebP, dropping from **1.6 MB to 24 KB (-99%)**.

### 4. Safety Guarantee: Zero-Size Regression (Step-Down Quality)
* If any initial WebP output is larger than the original asset (e.g. `Case-Study_Slider`: 321 KB WebP vs 309 KB PNG):
  * The engine steps down through fallback quality tiers: `90% -> 85% -> 80% -> 75% -> 70% -> 65% -> 60%`.
  * As soon as WebP size < Original size, that tier is committed (e.g. `Case-Study_Slider` → **217.8 KB, -30%**).
  * If even at minimum quality floor WebP ≥ Original, the image is skipped (`skipIfNoSavings: true`).

---

## 7. WordPress Deploy Package & Delivery

When optimization finishes, `WriteDeployPackage` produces a self-contained deploy folder (`speedmap-webp-YYYYMMDD-HHMMSS/`):

```
speedmap-webp-20260902-171531/
├── images/
│   ├── 001/optimized.webp
│   ├── 002/optimized.webp
│   └── ... (712 converted WebP images)
├── manifest.json         # Complete JSON map of source URLs, webpRel paths, sizes, and dimensions
├── apply.php             # Atomic zero-downtime deployment script for WordPress staging/prod
├── rollback.php          # 1-click instant rollback script to restore original attachments
├── compare.html          # Interactive visual before/after comparison tool (uses remote originals)
└── render-report.html    # Standalone diagnostic report with performance statistics
```

* **No Duplicate Originals in Package**: Originals are loaded on demand from live URLs in `compare.html`, keeping the package weight at **~46 MB instead of ~400 MB (-87% transfer reduction)**.

---

## 8. Export History & Automated Regression Tracking

Every export is recorded in `~/.speedmap/exports.json`:

```json
[
  {
    "id": "speedmap-webp-20260902-171531",
    "domain": "https://infuse.com/sitemap.xml",
    "timestamp": "2026-09-02T14:15:33Z",
    "formattedTime": "02.09.2026 17:15:33",
    "packageDir": "/path/to/speedmap-webp-20260902-171531",
    "manifestPath": "/path/to/speedmap-webp-20260902-171531/manifest.json",
    "imageCount": 712,
    "originalBytes": 344131580,
    "optimizedBytes": 48863616,
    "savingsPercent": 85.8,
    "existsOnDisk": true
  }
]
```

* **UI Regression Tab (`📈 Регресії`)**: Compares any two on-disk packages or scan runs:
  * 🔴 **Regressed files**: images whose WebP size increased by > 5 KB.
  * 🟢 **Improved files**: images whose WebP size decreased by > 5 KB.
  * ⚪ **Identical / Neutral**: images within ± 5 KB.
  * **Net Delta**: instant calculation of overall website weight reduction.

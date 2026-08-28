package scanner

import (
	"fmt"
)

// WebVitalsCollectorJS returns both WebVitals metrics and granular diagnostic insights
const WebVitalsCollectorJS = `
new Promise(async (resolve) => {
    const autoScrollEnabled = %t;
    const data = {
        metrics: {
            ttfb: 0,
            fcp: 0,
            lcp: 0,
            cls: 0,
            tbt: 0
        },
        diagnostics: {
            dnsTime: 0,
            tcpTime: 0,
            tlsTime: 0,
            serverProcessing: 0,
            renderBlockingCount: 0,
            renderBlockingFiles: [],
            lcpElement: '',
            lcpUrl: '',
            shiftCount: 0,
            shiftCauses: [],
            longTasksCount: 0,
            maxLongTaskMs: 0,
            slowestResources: [],
            largestImages: [],
            fonts: [],
            iframes: [],
            forms: [],
            categories: {}
        }
    };

    const formatBytes = (bytes) => {
        if (!bytes || bytes <= 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    };

    // Auto-scroll down the page to trigger lazy loading if enabled in settings
    if (autoScrollEnabled) {
        try {
            await new Promise((res) => {
                let totalHeight = 0;
                const distance = 400;
                const timer = setInterval(() => {
                    const scrollHeight = Math.max(document.body.scrollHeight || 0, document.documentElement.scrollHeight || 0);
                    window.scrollBy(0, distance);
                    totalHeight += distance;

                    if (totalHeight >= scrollHeight || totalHeight >= 10000) {
                        clearInterval(timer);
                        window.scrollTo(0, 0);
                        setTimeout(res, 350); // Allow brief pause for images and network requests to complete
                    }
                }, 60);
            });
        } catch (e) {}
    }

    // 1. Navigation Timing & Network breakdown
    try {
        const navEntries = performance.getEntriesByType('navigation');
        if (navEntries.length > 0) {
            const nav = navEntries[0];
            if (nav.domainLookupEnd > 0 && nav.domainLookupStart > 0) {
                data.diagnostics.dnsTime = Math.max(0, nav.domainLookupEnd - nav.domainLookupStart);
            }
            if (nav.connectEnd > 0 && nav.connectStart > 0) {
                data.diagnostics.tcpTime = Math.max(0, nav.connectEnd - nav.connectStart);
            }
            if (nav.secureConnectionStart > 0 && nav.connectEnd > 0) {
                data.diagnostics.tlsTime = Math.max(0, nav.connectEnd - nav.secureConnectionStart);
            }
            if (nav.responseStart > 0 && nav.requestStart > 0 && nav.responseStart >= nav.requestStart) {
                data.diagnostics.serverProcessing = Math.max(0, nav.responseStart - nav.requestStart);
                data.metrics.ttfb = data.diagnostics.serverProcessing;
            } else if (nav.responseStart > 0) {
                data.metrics.ttfb = nav.responseStart;
            }
        }
    } catch (e) {}

    // 2. FCP Paint & Render Blocking Resources
    try {
        const paintEntries = performance.getEntriesByType('paint');
        for (const entry of paintEntries) {
            if (entry.name === 'first-contentful-paint') {
                data.metrics.fcp = entry.startTime;
                break;
            } else if (entry.name === 'first-paint' && !data.metrics.fcp) {
                data.metrics.fcp = entry.startTime;
            }
        }

        const resources = performance.getEntriesByType('resource');
        const blocking = resources.filter(r => r.renderBlockingStatus === 'blocking' || (r.initiatorType === 'css' && r.duration > 100));
        data.diagnostics.renderBlockingCount = blocking.length;
        data.diagnostics.renderBlockingFiles = blocking.slice(0, 3).map(r => {
            try {
                return new URL(r.name).pathname.split('/').pop() || r.name;
            } catch(e) {
                return r.name;
            }
        });
    } catch (e) {}

    // 3. Observers for LCP, CLS, TBT
    let lcpValue = 0;
    let clsValue = 0;
    let longTaskCount = 0;
    let maxLongTask = 0;
    let totalLongTaskDuration = 0;
    let shiftCount = 0;

    try {
        const lcpObserver = new PerformanceObserver((entryList) => {
            const entries = entryList.getEntries();
            if (entries.length > 0) {
                const last = entries[entries.length - 1];
                lcpValue = last.startTime;
                if (last.element) {
                    const tag = last.element.tagName ? last.element.tagName.toLowerCase() : 'element';
                    const idStr = last.element.id ? '#' + last.element.id : '';
                    const classStr = last.element.className && typeof last.element.className === 'string' ? '.' + last.element.className.trim().split(/\s+/)[0] : '';
                    data.diagnostics.lcpElement = '<' + tag + idStr + classStr + '>';
                    data.diagnostics.lcpUrl = last.url || last.element.src || '';
                }
            }
        });
        lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
    } catch (e) {}

    try {
        const clsObserver = new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                if (!entry.hadRecentInput) {
                    clsValue += entry.value;
                    shiftCount++;
                }
            }
        });
        clsObserver.observe({ type: 'layout-shift', buffered: true });
    } catch (e) {}

    try {
        const longTaskObserver = new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                longTaskCount++;
                if (entry.duration > maxLongTask) {
                    maxLongTask = entry.duration;
                }
                if (entry.duration > 50) {
                    totalLongTaskDuration += (entry.duration - 50);
                }
            }
        });
        longTaskObserver.observe({ type: 'longtask', buffered: true });

        const existingLong = performance.getEntriesByType('longtask');
        for (const entry of existingLong) {
            longTaskCount++;
            if (entry.duration > maxLongTask) maxLongTask = entry.duration;
            if (entry.duration > 50) totalLongTaskDuration += (entry.duration - 50);
        }
    } catch (e) {}

    // Check for images without explicit dimensions (CLS causes)
    try {
        const imgs = Array.from(document.querySelectorAll('img'));
        const unsized = imgs.filter(i => !i.getAttribute('width') || !i.getAttribute('height'));
        if (unsized.length > 0) {
            data.diagnostics.shiftCauses.push(unsized.length + ' зображень без вказаних розмірів width/height');
        }
    } catch (e) {}

    // 4. Slowest Resources Breakdown
    try {
        const resources = performance.getEntriesByType('resource');
        const slow = resources
            .filter(r => r.duration > 150)
            .sort((a, b) => b.duration - a.duration)
            .slice(0, 5)
            .map(r => {
                let nameStr = r.name;
                try {
                    const urlObj = new URL(r.name);
                    nameStr = urlObj.pathname.split('/').pop() || urlObj.hostname;
                } catch(err) {}
                return {
                    name: nameStr,
                    duration: Math.round(r.duration),
                    type: r.initiatorType || 'resource'
                };
            });
        data.diagnostics.slowestResources = slow;
    } catch (e) {}

    // 5. Images Extraction & Diagnostics (Complete & Unlimited Extraction)
    try {
        const getImgFormat = (url) => {
            if (!url) return 'unknown';
            const cleanUrl = url.split('?')[0].toLowerCase();
            if (cleanUrl.endsWith('.png')) return 'png';
            if (cleanUrl.endsWith('.jpg') || cleanUrl.endsWith('.jpeg')) return 'jpg';
            if (cleanUrl.endsWith('.webp')) return 'webp';
            if (cleanUrl.endsWith('.avif')) return 'avif';
            if (cleanUrl.endsWith('.svg')) return 'svg';
            if (cleanUrl.endsWith('.gif')) return 'gif';
            return 'other';
        };

        const resources = performance.getEntriesByType('resource');
        const imageMap = new Map();
        const domImgMap = new Map();

        // A. Extract DOM <img> attributes and accurate display/rendered dimensions
        const imgElements = Array.from(document.querySelectorAll('img'));
        imgElements.forEach(img => {
            const src = img.currentSrc || img.src || img.getAttribute('data-src') || img.getAttribute('data-lazy-src');
            if (src && !src.startsWith('data:')) {
                let fullUrl = src;
                try { fullUrl = new URL(src, document.baseURI).href; } catch(e){}
                
                const rect = img.getBoundingClientRect();
                const renderedW = Math.round(rect.width || img.clientWidth || img.offsetWidth || 0);
                const renderedH = Math.round(rect.height || img.clientHeight || img.offsetHeight || 0);
                const naturalW = img.naturalWidth || img.width || 0;
                const naturalH = img.naturalHeight || img.height || 0;

                const prev = domImgMap.get(fullUrl);
                if (prev) {
                    domImgMap.set(fullUrl, {
                        width: naturalW || prev.width,
                        height: naturalH || prev.height,
                        naturalWidth: naturalW || prev.naturalWidth,
                        naturalHeight: naturalH || prev.naturalHeight,
                        renderedWidth: Math.max(prev.renderedWidth || 0, renderedW),
                        renderedHeight: Math.max(prev.renderedHeight || 0, renderedH),
                        isLazy: prev.isLazy || (img.getAttribute('loading') === 'lazy' || !!img.getAttribute('data-src')),
                        alt: img.getAttribute('alt') || prev.alt || ''
                    });
                } else {
                    domImgMap.set(fullUrl, {
                        width: naturalW,
                        height: naturalH,
                        naturalWidth: naturalW,
                        naturalHeight: naturalH,
                        renderedWidth: renderedW,
                        renderedHeight: renderedH,
                        isLazy: img.getAttribute('loading') === 'lazy' || !!img.getAttribute('data-src'),
                        alt: img.getAttribute('alt') || ''
                    });
                }
            }
        });

        // B. Extract Network Performance Resources
        resources.forEach(r => {
            const isImgType = r.initiatorType === 'img' || r.initiatorType === 'image' || r.initiatorType === 'css' || r.initiatorType === 'picture';
            const isImgExt = /\.(jpg|jpeg|png|webp|avif|gif|svg|ico|bmp)(\?.*)?$/i.test(r.name);
            if (isImgType || isImgExt) {
                const size = r.transferSize || r.encodedBodySize || r.decodedBodySize || 0;
                const domInfo = domImgMap.get(r.name) || { width: 0, height: 0, naturalWidth: 0, naturalHeight: 0, renderedWidth: 0, renderedHeight: 0, isLazy: false, alt: '' };
                imageMap.set(r.name, {
                    url: r.name,
                    transferSize: size,
                    encodedSize: r.encodedBodySize || 0,
                    duration: Math.round(r.duration || 0),
                    width: domInfo.width || domInfo.naturalWidth || 0,
                    height: domInfo.height || domInfo.naturalHeight || 0,
                    naturalWidth: domInfo.naturalWidth || domInfo.width || 0,
                    naturalHeight: domInfo.naturalHeight || domInfo.height || 0,
                    renderedWidth: domInfo.renderedWidth || 0,
                    renderedHeight: domInfo.renderedHeight || 0,
                    formattedSize: formatBytes(size),
                    format: getImgFormat(r.name),
                    isLazy: domInfo.isLazy,
                    alt: domInfo.alt,
                    isLCP: data.diagnostics.lcpUrl === r.name
                });
            }
        });

        // C. Add DOM <img> elements not in performance timings
        imgElements.forEach(img => {
            const src = img.currentSrc || img.src || img.getAttribute('data-src');
            if (src && !src.startsWith('data:')) {
                let fullUrl = src;
                try { fullUrl = new URL(src, document.baseURI).href; } catch(e){}
                if (!imageMap.has(fullUrl)) {
                    const domInfo = domImgMap.get(fullUrl) || { width: 0, height: 0, naturalWidth: 0, naturalHeight: 0, renderedWidth: 0, renderedHeight: 0, isLazy: false, alt: '' };
                    imageMap.set(fullUrl, {
                        url: fullUrl,
                        transferSize: 0,
                        encodedSize: 0,
                        duration: 0,
                        width: domInfo.width || domInfo.naturalWidth || 0,
                        height: domInfo.height || domInfo.naturalHeight || 0,
                        naturalWidth: domInfo.naturalWidth || domInfo.width || 0,
                        naturalHeight: domInfo.naturalHeight || domInfo.height || 0,
                        renderedWidth: domInfo.renderedWidth || 0,
                        renderedHeight: domInfo.renderedHeight || 0,
                        formattedSize: '0 B',
                        format: getImgFormat(fullUrl),
                        isLazy: domInfo.isLazy,
                        alt: domInfo.alt,
                        isLCP: data.diagnostics.lcpUrl === fullUrl
                    });
                }
            }
        });

        // D. Extract <picture> <source srcset="..."> and <img srcset="...">
        const pictureSources = Array.from(document.querySelectorAll('picture source, img[srcset]'));
        pictureSources.forEach(source => {
            const srcset = source.getAttribute('srcset') || source.getAttribute('data-srcset');
            if (srcset) {
                srcset.split(',').forEach(part => {
                    const rawUrl = part.trim().split(/\s+/)[0];
                    if (rawUrl && !rawUrl.startsWith('data:')) {
                        let fullUrl = rawUrl;
                        try { fullUrl = new URL(rawUrl, document.baseURI).href; } catch(e){}
                        if (!imageMap.has(fullUrl)) {
                            imageMap.set(fullUrl, {
                                url: fullUrl,
                                transferSize: 0,
                                encodedSize: 0,
                                duration: 0,
                                width: 0,
                                height: 0,
                                naturalWidth: 0,
                                naturalHeight: 0,
                                renderedWidth: 0,
                                renderedHeight: 0,
                                formattedSize: '0 B',
                                format: getImgFormat(fullUrl),
                                isLazy: source.getAttribute('loading') === 'lazy' || !!source.getAttribute('data-srcset'),
                                alt: '',
                                isLCP: data.diagnostics.lcpUrl === fullUrl
                            });
                        }
                    }
                });
            }
        });

        const imageList = Array.from(imageMap.values())
            .sort((a, b) => (b.transferSize - a.transferSize) || (b.duration - a.duration));

        data.diagnostics.largestImages = imageList;
    } catch (e) {}

    // 6. Font Usage & Resource Extraction
    try {
        const fontMap = new Map();
        const resources = performance.getEntriesByType('resource');

        // Extract font resource timings
        resources.forEach(r => {
            const isFontType = r.initiatorType === 'font';
            const isFontExt = /\.(woff2|woff|ttf|otf|eot)(\?.*)?$/i.test(r.name) || r.name.includes('fonts.gstatic.com') || r.name.includes('use.typekit.net');
            if (isFontType || isFontExt) {
                let familyName = '';
                try {
                    const u = new URL(r.name);
                    const filename = u.pathname.split('/').pop() || '';
                    familyName = filename.split('.')[0].replace(/[-_]/g, ' ');
                } catch(err) {
                    familyName = 'Web Font';
                }
                const size = r.transferSize || r.encodedBodySize || r.decodedBodySize || 0;
                let fontType = 'woff2';
                if (r.name.includes('.woff2')) fontType = 'woff2';
                else if (r.name.includes('.woff')) fontType = 'woff';
                else if (r.name.includes('.ttf')) fontType = 'ttf';
                else if (r.name.includes('.otf')) fontType = 'otf';
                else if (r.name.includes('gstatic.com') || r.name.includes('googleapis.com')) fontType = 'google-font';

                fontMap.set(familyName || r.name, {
                    family: familyName || 'Font Resource',
                    url: r.name,
                    type: fontType,
                    transferSize: size,
                    duration: Math.round(r.duration || 0)
                });
            }
        });

        // Extract document fonts API families
        if (document.fonts && document.fonts.forEach) {
            document.fonts.forEach(f => {
                const familyName = (f.family || '').replace(/["']/g, '').trim();
                if (familyName && !fontMap.has(familyName)) {
                    fontMap.set(familyName, {
                        family: familyName,
                        url: '',
                        type: f.status === 'loaded' ? 'loaded-font' : 'css-font',
                        transferSize: 0,
                        duration: 0
                    });
                }
            });
        }

        data.diagnostics.fonts = Array.from(fontMap.values()).slice(0, 8);
    } catch (e) {}

    // 7. Iframe Extraction (DOM + network timing → detect "missed" during load)
    try {
        const resources = performance.getEntriesByType('resource');
        const timingByUrl = new Map();
        resources.forEach(r => {
            const isIframe = r.initiatorType === 'iframe' || r.initiatorType === 'subdocument';
            if (isIframe) {
                timingByUrl.set(r.name, {
                    duration: Math.round(r.duration || 0),
                    transferSize: r.transferSize || r.encodedBodySize || 0
                });
            }
        });

        const iframeList = [];
        Array.from(document.querySelectorAll('iframe')).forEach((el, idx) => {
            const rawSrc = el.getAttribute('src') || el.src || el.getAttribute('data-src') || '';
            let fullUrl = '';
            if (rawSrc && !rawSrc.startsWith('data:') && !rawSrc.startsWith('javascript:')) {
                try { fullUrl = new URL(rawSrc, document.baseURI).href; } catch (e) { fullUrl = rawSrc; }
            }

            let duration = 0;
            let transferSize = 0;
            let loadedDuringScan = false;
            if (fullUrl) {
                let timing = timingByUrl.get(fullUrl);
                if (!timing) {
                    const match = resources.find(r => r.name === fullUrl || (fullUrl && r.name.indexOf(fullUrl.split('?')[0]) === 0));
                    if (match) {
                        timing = {
                            duration: Math.round(match.duration || 0),
                            transferSize: match.transferSize || match.encodedBodySize || 0
                        };
                    }
                }
                if (timing) {
                    duration = timing.duration;
                    transferSize = timing.transferSize;
                    loadedDuringScan = true;
                }
            }

            let inViewport = false;
            try {
                const rect = el.getBoundingClientRect();
                inViewport = rect.top < (window.innerHeight || 0) && rect.bottom > 0 &&
                    rect.left < (window.innerWidth || 0) && rect.right > 0 &&
                    rect.width > 0 && rect.height > 0;
            } catch (e) {}

            iframeList.push({
                src: fullUrl || (rawSrc ? rawSrc : ('(empty-src#' + idx + ')')),
                title: el.getAttribute('title') || el.getAttribute('name') || el.id || '',
                width: parseInt(el.getAttribute('width'), 10) || Math.round(el.getBoundingClientRect().width) || 0,
                height: parseInt(el.getAttribute('height'), 10) || Math.round(el.getBoundingClientRect().height) || 0,
                isLazy: el.getAttribute('loading') === 'lazy' || !!el.getAttribute('data-src'),
                loadedDuringScan: loadedDuringScan,
                inViewport: inViewport,
                sandbox: el.hasAttribute('sandbox'),
                duration: duration,
                transferSize: transferSize,
                formattedSize: formatBytes(transferSize)
            });
        });

        data.diagnostics.iframes = iframeList;
    } catch (e) {}

    // 8. Form Extraction (DOM + Engine Classification + Captcha + Fields)
    try {
        const formList = [];
        const seenFormElements = new Set();

        const detectEngine = (formEl) => {
            const markup = formEl.outerHTML || '';
            const action = formEl.getAttribute('action') || '';
            const className = formEl.className || '';
            const id = formEl.id || '';

            if (className.includes('wpcf7') || formEl.closest('.wpcf7') || markup.includes('wpcf7') || formEl.querySelector('[name^="_wpcf7"]')) {
                return 'contact-form-7';
            }
            if (markup.includes('greenhouse.io') || formEl.querySelector('[name="jobs_id"]') || className.includes('greenhouse') || id.includes('application_form') || formEl.closest('.join-position-hiring-flex__right--form')) {
                return 'greenhouse';
            }
            if (className.includes('pardot') || markup.includes('pardot_form_send') || action.includes('pardot.com') || action.includes('go.pardot.com')) {
                return 'pardot';
            }
            if (className.includes('hs-form') || className.includes('hbspt') || formEl.querySelector('[data-form-id]')) {
                return 'hubspot';
            }
            if (className.includes('gform_wrapper') || id.startsWith('gform_')) {
                return 'gravity-forms';
            }
            if (className.includes('wpforms-form')) {
                return 'wpforms';
            }
            if (id.includes('mc-embedded') || action.includes('list-manage.com')) {
                return 'mailchimp';
            }
            return 'native-html';
        };

        const detectCaptcha = (formEl) => {
            const hasRecaptchaV3 = !!formEl.querySelector('input[name="_wpcf7_recaptcha_response"]') ||
                                  !!formEl.querySelector('input[name="g-recaptcha-response"]') ||
                                  !!document.querySelector('.grecaptcha-badge') ||
                                  (typeof window.grecaptcha !== 'undefined');

            const hasRecaptchaV2 = !!formEl.querySelector('.g-recaptcha') ||
                                  !!formEl.querySelector('iframe[src*="google.com/recaptcha/api2"]') ||
                                  !!formEl.querySelector('[data-sitekey]');

            const hasTurnstile = !!formEl.querySelector('.cf-turnstile') ||
                                !!formEl.querySelector('iframe[src*="challenges.cloudflare.com"]') ||
                                !!formEl.querySelector('[data-turnstile-sitekey]');

            const hasHcaptcha = !!formEl.querySelector('.h-captcha') ||
                               !!formEl.querySelector('iframe[src*="hcaptcha.com"]');

            if (hasRecaptchaV3) {
                let siteKey = '';
                const scriptEl = Array.from(document.querySelectorAll('script[src*="recaptcha"]')).find(s => s.src.includes('render='));
                if (scriptEl) {
                    try { siteKey = new URL(scriptEl.src).searchParams.get('render') || ''; } catch(e) {}
                }
                return { type: 'recaptcha-v3', siteKey: siteKey, action: 'contact_form', isActive: true };
            }
            if (hasRecaptchaV2) {
                const el = formEl.querySelector('[data-sitekey]') || document.querySelector('[data-sitekey]');
                return { type: 'recaptcha-v2', siteKey: el ? el.getAttribute('data-sitekey') : '', action: '', isActive: true };
            }
            if (hasTurnstile) {
                const el = formEl.querySelector('[data-turnstile-sitekey]') || formEl.querySelector('.cf-turnstile');
                return { type: 'turnstile', siteKey: el ? (el.getAttribute('data-turnstile-sitekey') || el.getAttribute('data-sitekey') || '') : '', action: '', isActive: true };
            }
            if (hasHcaptcha) {
                const el = formEl.querySelector('.h-captcha');
                return { type: 'hcaptcha', siteKey: el ? el.getAttribute('data-sitekey') : '', action: '', isActive: true };
            }
            return { type: 'none', siteKey: '', action: '', isActive: false };
        };

        const formNodes = Array.from(document.querySelectorAll('form, .wpcf7, .greenhouse-form, .join-position-hiring-flex__right--form'));
        
        formNodes.forEach((node, formIdx) => {
            const formEl = node.tagName && node.tagName.toLowerCase() === 'form' ? node : (node.querySelector('form') || node);
            if (seenFormElements.has(formEl)) return;
            seenFormElements.add(formEl);

            const engine = detectEngine(formEl);
            const captcha = detectCaptcha(formEl);
            
            let title = '';
            const headingEl = formEl.closest('section, div, footer')?.querySelector('h1, h2, h3, .footer-form__title, .hero-banner__form-popup-title') ||
                              formEl.querySelector('h1, h2, h3, legend, .form-title');
            if (headingEl) {
                title = headingEl.textContent.trim().slice(0, 100);
            }
            if (!title) {
                title = formEl.getAttribute('aria-label') || formEl.getAttribute('name') || formEl.id || (engine + ' #' + (formIdx + 1));
            }

            const inputElements = Array.from(formEl.querySelectorAll('input, textarea, select, button[type="submit"], input[type="submit"]'));
            const fields = [];
            const hiddenTokens = {};
            let hasFileUpload = false;
            let allowedFileTypes = '';

            inputElements.forEach(inp => {
                const tag = inp.tagName.toLowerCase();
                let type = (inp.getAttribute('type') || (tag === 'textarea' ? 'textarea' : (tag === 'select' ? 'select' : 'text'))).toLowerCase();
                const name = inp.getAttribute('name') || inp.id || '';
                const placeholder = inp.getAttribute('placeholder') || '';
                
                let label = '';
                if (inp.id) {
                    const lblEl = formEl.querySelector('label[for="' + inp.id + '"]');
                    if (lblEl) label = lblEl.textContent.trim();
                }
                if (!label && inp.closest('label')) {
                    label = inp.closest('label').textContent.trim();
                }
                if (!label) {
                    label = placeholder || inp.getAttribute('aria-label') || name;
                }

                const isRequired = inp.hasAttribute('required') || inp.getAttribute('aria-required') === 'true' || (name.includes('*') || label.includes('*'));
                const validationPattern = inp.getAttribute('pattern') || '';
                const accept = inp.getAttribute('accept') || '';
                const value = inp.value || inp.getAttribute('value') || '';

                if (type === 'file') {
                    hasFileUpload = true;
                    if (accept) allowedFileTypes = accept;
                }

                if (type === 'hidden') {
                    if (name) hiddenTokens[name] = value;
                }

                if (name === '_wpcf7_recaptcha_response' || name === 'g-recaptcha-response') {
                    return;
                }

                fields.push({
                    name: name || ('(unnamed_' + fields.length + ')'),
                    label: label.slice(0, 80),
                    type: type,
                    isRequired: isRequired,
                    validationPattern: validationPattern,
                    accept: accept,
                    value: type === 'hidden' ? value : ''
                });
            });

            let inViewport = false;
            try {
                const rect = formEl.getBoundingClientRect();
                inViewport = rect.top < (window.innerHeight || 0) && rect.bottom > 0 &&
                    rect.left < (window.innerWidth || 0) && rect.right > 0 &&
                    rect.width > 0 && rect.height > 0;
            } catch(e) {}

            formList.push({
                id: formEl.id || ('form-' + (formIdx + 1)),
                title: title,
                engine: engine,
                method: (formEl.getAttribute('method') || 'POST').toUpperCase(),
                action: formEl.getAttribute('action') || (formEl.getAttribute('data-action') || ''),
                fields: fields,
                fieldCount: fields.length,
                hasFileUpload: hasFileUpload,
                allowedFileTypes: allowedFileTypes,
                captcha: captcha,
                hiddenTokens: hiddenTokens,
                inViewport: inViewport
            });
        });

        data.diagnostics.forms = formList;
    } catch(e) {}

    // 1.0s observation window
    setTimeout(() => {
        data.metrics.lcp = lcpValue || data.metrics.fcp || data.metrics.ttfb || 0;
        data.metrics.fcp = data.metrics.fcp || (data.metrics.lcp ? Math.min(data.metrics.fcp || data.metrics.lcp, data.metrics.lcp) : data.metrics.ttfb);
        data.metrics.cls = clsValue || 0;
        data.metrics.tbt = totalLongTaskDuration || 0;
        data.diagnostics.longTasksCount = longTaskCount;
        data.diagnostics.maxLongTaskMs = maxLongTask;
        data.diagnostics.shiftCount = shiftCount;

        resolve(data);
    }, 1000);
});
`

type ScanEvalResult struct {
	Metrics     WebVitals       `json:"metrics"`
	Diagnostics PageDiagnostics `json:"diagnostics"`
}

func BuildCategoryDiagnostics(m WebVitals, grades DetailedGrades, diag PageDiagnostics) map[string]CategoryDiagnostic {
	cats := make(map[string]CategoryDiagnostic)

	// 1. TTFB Category
	ttfbDiag := CategoryDiagnostic{
		Category: "ttfb",
		Title:    "TTFB (Час відповіді сервера)",
		Status:   grades.TTFB.Status,
		Summary:  fmt.Sprintf("Початковий час відповіді сервера становить %s.", grades.TTFB.Formatted),
	}
	if diag.ServerProcessing > 0 {
		ttfbDiag.Details = append(ttfbDiag.Details, fmt.Sprintf("Час обробки на бекенді: %.0fms", diag.ServerProcessing))
	}
	if diag.DNSTime > 0 {
		ttfbDiag.Details = append(ttfbDiag.Details, fmt.Sprintf("DNS Lookup: %.0fms", diag.DNSTime))
	}
	if diag.TCPTime > 0 {
		ttfbDiag.Details = append(ttfbDiag.Details, fmt.Sprintf("TCP Handshake / TLS: %.0fms", diag.TCPTime))
	}
	if grades.TTFB.Status != "good" {
		ttfbDiag.Fixes = append(ttfbDiag.Fixes,
			"Увімкніть кешування на рівні сервера (OPcache, Redis, Nginx FastCGI Cache або Cloudflare CDN).",
			"Оптимізуйте повільні запити до бази даних (SQL indexes) та скрипти бекенду.",
		)
	} else {
		ttfbDiag.Fixes = append(ttfbDiag.Fixes, "Сервер відповідає швидко. Показник у межах норми.")
	}
	cats["ttfb"] = ttfbDiag

	// 2. FCP Category
	fcpDiag := CategoryDiagnostic{
		Category: "fcp",
		Title:    "FCP (Перше малювання вмісту)",
		Status:   grades.FCP.Status,
		Summary:  fmt.Sprintf("Перший видимий елемент з'являється через %s після запиту.", grades.FCP.Formatted),
	}
	if diag.RenderBlockingCount > 0 {
		fcpDiag.Details = append(fcpDiag.Details, fmt.Sprintf("Виявлено %d стилів/скриптів, що блокують рендеринг", diag.RenderBlockingCount))
		for _, f := range diag.RenderBlockingFiles {
			fcpDiag.Details = append(fcpDiag.Details, fmt.Sprintf("Блокуючий ресурс: %s", f))
		}
	}
	if grades.FCP.Status != "good" {
		fcpDiag.Fixes = append(fcpDiag.Fixes,
			"Додайте атрибути 'defer' або 'async' для всіх некритичних скриптів JavaScript.",
			"Скоротіть обсяг критичного CSS або додайте inline critical CSS для швидкого відображення першого екрану.",
		)
	} else {
		fcpDiag.Fixes = append(fcpDiag.Fixes, "Перше малювання відбувається без затримок.")
	}
	cats["fcp"] = fcpDiag

	// 3. LCP Category
	lcpDiag := CategoryDiagnostic{
		Category: "lcp",
		Title:    "LCP (Найбільший елемент макету)",
		Status:   grades.LCP.Status,
		Summary:  fmt.Sprintf("Найбільший контентний елемент макету відмальовується за %s.", grades.LCP.Formatted),
	}
	if diag.LCPElement != "" {
		lcpDiag.Details = append(lcpDiag.Details, fmt.Sprintf("Головний LCP елемент: %s", diag.LCPElement))
	}
	if diag.LCPUrl != "" {
		lcpDiag.Details = append(lcpDiag.Details, fmt.Sprintf("Ресурс зображення/шрифту: %s", diag.LCPUrl))
	}
	if grades.LCP.Status != "good" {
		lcpDiag.Fixes = append(lcpDiag.Fixes,
			"Конвертуйте зображення у сучасні формати WebP/AVIF та стисніть розмір.",
			"Додайте `<link rel='preload' as='image'>` або атрибут `fetchpriority='high'` для LCP зображення.",
		)
	} else {
		lcpDiag.Fixes = append(fcpDiag.Fixes, "Головний елемент сторінки завантажується оптимально.")
	}
	cats["lcp"] = lcpDiag

	// 4. CLS Category
	clsDiag := CategoryDiagnostic{
		Category: "cls",
		Title:    "CLS (Зсув макету сторінки)",
		Status:   grades.CLS.Status,
		Summary:  fmt.Sprintf("Сумарний зсув елементів макету становить %s.", grades.CLS.Formatted),
	}
	if diag.ShiftCount > 0 {
		clsDiag.Details = append(clsDiag.Details, fmt.Sprintf("Фіксовано %d зафіксованих зсувів елементів під час рендерингу", diag.ShiftCount))
	}
	for _, cause := range diag.ShiftCauses {
		clsDiag.Details = append(clsDiag.Details, cause)
	}
	if grades.CLS.Status != "good" {
		clsDiag.Fixes = append(clsDiag.Fixes,
			"Вкажіть точні атрибути `width` та `height` для всіх <img> та <iframe> елементів.",
			"Зарезервуйте фіксоване місце під динамічні баннери чи віджети через CSS `min-height`.",
		)
	} else {
		clsDiag.Fixes = append(clsDiag.Fixes, "Макет сторінки залишається стабільним без небажаних зсувів.")
	}
	cats["cls"] = clsDiag

	// 5. TBT Category
	tbtDiag := CategoryDiagnostic{
		Category: "tbt",
		Title:    "TBT (Сумарне блокування потоку)",
		Status:   grades.TBT.Status,
		Summary:  fmt.Sprintf("Головний потік JS заблоковано на %s.", grades.TBT.Formatted),
	}
	if diag.LongTasksCount > 0 {
		tbtDiag.Details = append(tbtDiag.Details, fmt.Sprintf("Кількість довготривалих задач (>50ms): %d", diag.LongTasksCount))
	}
	if diag.MaxLongTaskMs > 0 {
		tbtDiag.Details = append(tbtDiag.Details, fmt.Sprintf("Найдовша задача на головному потоці: %.0fms", diag.MaxLongTaskMs))
	}
	if grades.TBT.Status != "good" {
		tbtDiag.Fixes = append(tbtDiag.Fixes,
			"Оптимізуйте та розбийте важкі функції JavaScript за допомогою `requestIdleCallback` або `setTimeout`.",
			"Видаліть або відкладіть завантаження некритичних трекерів та важких JS-бібліотек.",
		)
	} else {
		tbtDiag.Fixes = append(tbtDiag.Fixes, "Головний потік JavaScript вільний для взаємодії з користувачем.")
	}
	cats["tbt"] = tbtDiag

	return cats
}

func GenerateRecommendations(m WebVitals, grades DetailedGrades, diag PageDiagnostics) []string {
	var recs []string

	if grades.TTFB.Status != "good" {
		diagText := ""
		if diag.ServerProcessing > 0 {
			diagText = fmt.Sprintf(" Час обробки сервером: %.0fms, DNS: %.0fms, TCP: %.0fms.", diag.ServerProcessing, diag.DNSTime, diag.TCPTime)
		}
		recs = append(recs, fmt.Sprintf("Високий TTFB (%s): Повільна відповідь сервера.%s Рекомендація: оптимізуйте серверні запити, використайте CDN або HTTP кешування.", grades.TTFB.Formatted, diagText))
	}

	if grades.FCP.Status != "good" {
		recs = append(recs, fmt.Sprintf("Повільний FCP (%s): Перший вміст з'являється із затримкою. Рекомендація: приберіть стилі/скрипти, що блокують рендеринг.", grades.FCP.Formatted))
	}

	if grades.LCP.Status != "good" {
		lcpInfo := ""
		if diag.LCPElement != "" {
			lcpInfo = fmt.Sprintf(" Головний LCP елемент: %s.", diag.LCPElement)
		}
		if diag.LCPUrl != "" {
			lcpInfo += fmt.Sprintf(" Зображення/Ресурс: %s.", diag.LCPUrl)
		}
		recs = append(recs, fmt.Sprintf("Критичний LCP (%s): Найбільший елемент завантажується занадто довго.%s Рекомендація: стисніть зображення (WebP/AVIF), додайте `fetchpriority='high'` або зробіть preload.", grades.LCP.Formatted, lcpInfo))
	}

	if grades.CLS.Status != "good" {
		recs = append(recs, fmt.Sprintf("Високий CLS (%s): Зміщення макету під час завантаження. Рекомендація: додайте чіткі width/height для зображень та баннерів.", grades.CLS.Formatted))
	}

	if grades.TBT.Status != "good" {
		tasksText := ""
		if diag.LongTasksCount > 0 {
			tasksText = fmt.Sprintf(" Виявлено %d довготривалих задач (>50ms) на головному потоці.", diag.LongTasksCount)
		}
		recs = append(recs, fmt.Sprintf("Високий TBT (%s): Заблоковано головний потік JavaScript.%s Рекомендація: розбийте важкі JS-функції та видаліть невикористовуваний JS.", grades.TBT.Formatted, tasksText))
	}

	if len(diag.SlowestResources) > 0 {
		slowText := "Повільні ресурси сторінки: "
		for i, res := range diag.SlowestResources {
			if i > 0 {
				slowText += ", "
			}
			slowText += fmt.Sprintf("%s (%s, %.0fms)", res.Name, res.Type, res.Duration)
		}
		recs = append(recs, slowText)
	}

	if len(recs) == 0 {
		recs = append(recs, "Чудова продуктивність! Усі метрикі Core Web Vitals у межах зеленої зони Google.")
	}

	return recs
}

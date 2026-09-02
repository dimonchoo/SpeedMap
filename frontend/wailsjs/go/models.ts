export namespace analytics {
	
	export class AggregatedFont {
	    family: string;
	    url: string;
	    type: string;
	    occurrences: number;
	    percentage: number;
	    avgDurationMs: number;
	    transferSize: number;
	    formattedSize: string;
	    pageUrls: string[];
	
	    static createFrom(source: any = {}) {
	        return new AggregatedFont(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = source["family"];
	        this.url = source["url"];
	        this.type = source["type"];
	        this.occurrences = source["occurrences"];
	        this.percentage = source["percentage"];
	        this.avgDurationMs = source["avgDurationMs"];
	        this.transferSize = source["transferSize"];
	        this.formattedSize = source["formattedSize"];
	        this.pageUrls = source["pageUrls"];
	    }
	}
	export class AggregatedForm {
	    id: string;
	    title: string;
	    engine: string;
	    method: string;
	    action: string;
	    pageCount: number;
	    pages: string[];
	    fields: scanner.FormFieldDetail[];
	    fieldCount: number;
	    hasFileUpload: boolean;
	    allowedFileTypes?: string;
	    captcha: scanner.CaptchaDetail;
	    hiddenTokens?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new AggregatedForm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.engine = source["engine"];
	        this.method = source["method"];
	        this.action = source["action"];
	        this.pageCount = source["pageCount"];
	        this.pages = source["pages"];
	        this.fields = this.convertValues(source["fields"], scanner.FormFieldDetail);
	        this.fieldCount = source["fieldCount"];
	        this.hasFileUpload = source["hasFileUpload"];
	        this.allowedFileTypes = source["allowedFileTypes"];
	        this.captcha = this.convertValues(source["captcha"], scanner.CaptchaDetail);
	        this.hiddenTokens = source["hiddenTokens"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AggregatedIframe {
	    src: string;
	    title: string;
	    pageCount: number;
	    pages: string[];
	    occurrences: number;
	    loadedCount: number;
	    missedCount: number;
	    isLazy: boolean;
	    avgDurationMs: number;
	    maxTransferSize: number;
	    formattedSize: string;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new AggregatedIframe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.title = source["title"];
	        this.pageCount = source["pageCount"];
	        this.pages = source["pages"];
	        this.occurrences = source["occurrences"];
	        this.loadedCount = source["loadedCount"];
	        this.missedCount = source["missedCount"];
	        this.isLazy = source["isLazy"];
	        this.avgDurationMs = source["avgDurationMs"];
	        this.maxTransferSize = source["maxTransferSize"];
	        this.formattedSize = source["formattedSize"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class AggregatedImage {
	    url: string;
	    maxTransferSize: number;
	    formattedSize: string;
	    avgDurationMs: number;
	    pageCount: number;
	    pages: string[];
	    width: number;
	    height: number;
	    naturalWidth: number;
	    naturalHeight: number;
	    maxRenderedWidth: number;
	    maxRenderedHeight: number;
	    recommendedRetinaWidth: number;
	    recommendedRetinaHeight: number;
	    isOversized: boolean;
	    format: string;
	    isHeavy: boolean;
	    isLazy: boolean;
	    isLCP: boolean;
	    estimatedWebPSize: number;
	    estimatedWebPFormatted: string;
	    estimatedSavingsBytes: number;
	    estimatedSavingsFormatted: string;
	    estimatedSavingsPercent: number;
	
	    static createFrom(source: any = {}) {
	        return new AggregatedImage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.maxTransferSize = source["maxTransferSize"];
	        this.formattedSize = source["formattedSize"];
	        this.avgDurationMs = source["avgDurationMs"];
	        this.pageCount = source["pageCount"];
	        this.pages = source["pages"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.naturalWidth = source["naturalWidth"];
	        this.naturalHeight = source["naturalHeight"];
	        this.maxRenderedWidth = source["maxRenderedWidth"];
	        this.maxRenderedHeight = source["maxRenderedHeight"];
	        this.recommendedRetinaWidth = source["recommendedRetinaWidth"];
	        this.recommendedRetinaHeight = source["recommendedRetinaHeight"];
	        this.isOversized = source["isOversized"];
	        this.format = source["format"];
	        this.isHeavy = source["isHeavy"];
	        this.isLazy = source["isLazy"];
	        this.isLCP = source["isLCP"];
	        this.estimatedWebPSize = source["estimatedWebPSize"];
	        this.estimatedWebPFormatted = source["estimatedWebPFormatted"];
	        this.estimatedSavingsBytes = source["estimatedSavingsBytes"];
	        this.estimatedSavingsFormatted = source["estimatedSavingsFormatted"];
	        this.estimatedSavingsPercent = source["estimatedSavingsPercent"];
	    }
	}
	export class ResourceImpact {
	    name: string;
	    type: string;
	    occurrences: number;
	    avgDurationMs: number;
	    totalDurationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ResourceImpact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.occurrences = source["occurrences"];
	        this.avgDurationMs = source["avgDurationMs"];
	        this.totalDurationMs = source["totalDurationMs"];
	    }
	}
	export class SiteAnalytics {
	    totalPages: number;
	    healthScore: number;
	    statusCounts: Record<string, number>;
	    averageMetrics: Record<string, number>;
	    topResourceBottlenecks: ResourceImpact[];
	    largestImages: AggregatedImage[];
	    allImages: AggregatedImage[];
	    fontUsage: AggregatedFont[];
	    iframes: AggregatedIframe[];
	    forms: AggregatedForm[];
	    globalFixes: string[];
	    totalImagePayloadBytes: number;
	    totalImagePayloadFormatted: string;
	    totalImageCount: number;
	    heavyImagesCount: number;
	    oversizedImagesCount: number;
	    nonWebPCount: number;
	    missingLazyCount: number;
	    totalWebPSavingsBytes: number;
	    totalWebPSavingsFormatted: string;
	    formatBreakdown: Record<string, number>;
	    totalIframeCount: number;
	    missedIframeCount: number;
	    loadedIframeCount: number;
	    totalFormsCount: number;
	    pagesWithFormsCount: number;
	    captchaProtectedCount: number;
	    unprotectedFormsCount: number;
	    fileUploadFormsCount: number;
	    formEngineBreakdown: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new SiteAnalytics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalPages = source["totalPages"];
	        this.healthScore = source["healthScore"];
	        this.statusCounts = source["statusCounts"];
	        this.averageMetrics = source["averageMetrics"];
	        this.topResourceBottlenecks = this.convertValues(source["topResourceBottlenecks"], ResourceImpact);
	        this.largestImages = this.convertValues(source["largestImages"], AggregatedImage);
	        this.allImages = this.convertValues(source["allImages"], AggregatedImage);
	        this.fontUsage = this.convertValues(source["fontUsage"], AggregatedFont);
	        this.iframes = this.convertValues(source["iframes"], AggregatedIframe);
	        this.forms = this.convertValues(source["forms"], AggregatedForm);
	        this.globalFixes = source["globalFixes"];
	        this.totalImagePayloadBytes = source["totalImagePayloadBytes"];
	        this.totalImagePayloadFormatted = source["totalImagePayloadFormatted"];
	        this.totalImageCount = source["totalImageCount"];
	        this.heavyImagesCount = source["heavyImagesCount"];
	        this.oversizedImagesCount = source["oversizedImagesCount"];
	        this.nonWebPCount = source["nonWebPCount"];
	        this.missingLazyCount = source["missingLazyCount"];
	        this.totalWebPSavingsBytes = source["totalWebPSavingsBytes"];
	        this.totalWebPSavingsFormatted = source["totalWebPSavingsFormatted"];
	        this.formatBreakdown = source["formatBreakdown"];
	        this.totalIframeCount = source["totalIframeCount"];
	        this.missedIframeCount = source["missedIframeCount"];
	        this.loadedIframeCount = source["loadedIframeCount"];
	        this.totalFormsCount = source["totalFormsCount"];
	        this.pagesWithFormsCount = source["pagesWithFormsCount"];
	        this.captchaProtectedCount = source["captchaProtectedCount"];
	        this.unprotectedFormsCount = source["unprotectedFormsCount"];
	        this.fileUploadFormsCount = source["fileUploadFormsCount"];
	        this.formEngineBreakdown = source["formEngineBreakdown"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace cloud {
	
	export class DriveUploadResult {
	    fileId: string;
	    fileName: string;
	    webViewLink: string;
	    webContentLink: string;
	
	    static createFrom(source: any = {}) {
	        return new DriveUploadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileId = source["fileId"];
	        this.fileName = source["fileName"];
	        this.webViewLink = source["webViewLink"];
	        this.webContentLink = source["webContentLink"];
	    }
	}

}

export namespace config {
	
	export class CustomHeader {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomHeader(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class ScanConfig {
	    sitemapUrl: string;
	    concurrency: number;
	    heavyImageThresholdKB: number;
	    webpQuality: number;
	    pngWebPRatio: number;
	    jpgWebPRatio: number;
	    gifWebPRatio: number;
	    authUser: string;
	    authPass: string;
	    headers: CustomHeader[];
	    isMobile: boolean;
	    autoScroll: boolean;
	    timeoutSec: number;
	    gdriveClientID: string;
	    gdriveClientSecret: string;
	    adaptiveQuality?: boolean;
	    resizeToRetina?: boolean;
	    autoPruneHistory?: boolean;
	    historyRetentionRuns: number;
	    historyRetentionDays: number;
	    filterTrackingBeacons?: boolean;
	    excludedImagePatterns: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapUrl = source["sitemapUrl"];
	        this.concurrency = source["concurrency"];
	        this.heavyImageThresholdKB = source["heavyImageThresholdKB"];
	        this.webpQuality = source["webpQuality"];
	        this.pngWebPRatio = source["pngWebPRatio"];
	        this.jpgWebPRatio = source["jpgWebPRatio"];
	        this.gifWebPRatio = source["gifWebPRatio"];
	        this.authUser = source["authUser"];
	        this.authPass = source["authPass"];
	        this.headers = this.convertValues(source["headers"], CustomHeader);
	        this.isMobile = source["isMobile"];
	        this.autoScroll = source["autoScroll"];
	        this.timeoutSec = source["timeoutSec"];
	        this.gdriveClientID = source["gdriveClientID"];
	        this.gdriveClientSecret = source["gdriveClientSecret"];
	        this.adaptiveQuality = source["adaptiveQuality"];
	        this.resizeToRetina = source["resizeToRetina"];
	        this.autoPruneHistory = source["autoPruneHistory"];
	        this.historyRetentionRuns = source["historyRetentionRuns"];
	        this.historyRetentionDays = source["historyRetentionDays"];
	        this.filterTrackingBeacons = source["filterTrackingBeacons"];
	        this.excludedImagePatterns = source["excludedImagePatterns"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace history {
	
	export class RunComparison {
	    hasPrevious: boolean;
	    previousTime: string;
	    previousScore: number;
	    currentScore: number;
	    scoreDelta: number;
	    metricDeltas: Record<string, number>;
	    summaryStatus: string;
	    summaryText: string;
	
	    static createFrom(source: any = {}) {
	        return new RunComparison(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasPrevious = source["hasPrevious"];
	        this.previousTime = source["previousTime"];
	        this.previousScore = source["previousScore"];
	        this.currentScore = source["currentScore"];
	        this.scoreDelta = source["scoreDelta"];
	        this.metricDeltas = source["metricDeltas"];
	        this.summaryStatus = source["summaryStatus"];
	        this.summaryText = source["summaryText"];
	    }
	}

}

export namespace main {
	
	export class AnalyticsResult {
	    analytics: analytics.SiteAnalytics;
	    comparison: history.RunComparison;
	
	    static createFrom(source: any = {}) {
	        return new AnalyticsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.analytics = this.convertValues(source["analytics"], analytics.SiteAnalytics);
	        this.comparison = this.convertValues(source["comparison"], history.RunComparison);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace optimizer {
	
	export class ConversionResult {
	    url: string;
	    filename: string;
	    originalWidth?: number;
	    originalHeight?: number;
	    optimizedWidth?: number;
	    optimizedHeight?: number;
	    originalBytes: number;
	    originalFormatted: string;
	    optimizedBytes: number;
	    optimizedFormatted: string;
	    savingsBytes: number;
	    savingsFormatted: string;
	    savingsPercent: number;
	    qualityUsed: number;
	    isLossless: boolean;
	    adaptiveApplied: boolean;
	    originalDataBase64: string;
	    optimizedWebPBase64: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.originalWidth = source["originalWidth"];
	        this.originalHeight = source["originalHeight"];
	        this.optimizedWidth = source["optimizedWidth"];
	        this.optimizedHeight = source["optimizedHeight"];
	        this.originalBytes = source["originalBytes"];
	        this.originalFormatted = source["originalFormatted"];
	        this.optimizedBytes = source["optimizedBytes"];
	        this.optimizedFormatted = source["optimizedFormatted"];
	        this.savingsBytes = source["savingsBytes"];
	        this.savingsFormatted = source["savingsFormatted"];
	        this.savingsPercent = source["savingsPercent"];
	        this.qualityUsed = source["qualityUsed"];
	        this.isLossless = source["isLossless"];
	        this.adaptiveApplied = source["adaptiveApplied"];
	        this.originalDataBase64 = source["originalDataBase64"];
	        this.optimizedWebPBase64 = source["optimizedWebPBase64"];
	        this.error = source["error"];
	    }
	}

}

export namespace profiles {
	
	export class SiteProfile {
	    id: string;
	    name: string;
	    sitemapUrl: string;
	    config: config.ScanConfig;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    lastScannedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new SiteProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sitemapUrl = source["sitemapUrl"];
	        this.config = this.convertValues(source["config"], config.ScanConfig);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.lastScannedAt = this.convertValues(source["lastScannedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace scanner {
	
	export class CaptchaDetail {
	    type: string;
	    siteKey?: string;
	    action?: string;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CaptchaDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.siteKey = source["siteKey"];
	        this.action = source["action"];
	        this.isActive = source["isActive"];
	    }
	}
	export class CategoryDiagnostic {
	    category: string;
	    title: string;
	    status: string;
	    summary: string;
	    details: string[];
	    fixes: string[];
	
	    static createFrom(source: any = {}) {
	        return new CategoryDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.summary = source["summary"];
	        this.details = source["details"];
	        this.fixes = source["fixes"];
	    }
	}
	export class MetricGrade {
	    value: number;
	    formatted: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new MetricGrade(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.formatted = source["formatted"];
	        this.status = source["status"];
	    }
	}
	export class DetailedGrades {
	    ttfb: MetricGrade;
	    fcp: MetricGrade;
	    lcp: MetricGrade;
	    cls: MetricGrade;
	    tbt: MetricGrade;
	
	    static createFrom(source: any = {}) {
	        return new DetailedGrades(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ttfb = this.convertValues(source["ttfb"], MetricGrade);
	        this.fcp = this.convertValues(source["fcp"], MetricGrade);
	        this.lcp = this.convertValues(source["lcp"], MetricGrade);
	        this.cls = this.convertValues(source["cls"], MetricGrade);
	        this.tbt = this.convertValues(source["tbt"], MetricGrade);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DomainResolution {
	    domain: string;
	    ips: string[];
	    ip: string;
	    isCloudflare: boolean;
	    provider: string;
	    serverHeader: string;
	
	    static createFrom(source: any = {}) {
	        return new DomainResolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.ips = source["ips"];
	        this.ip = source["ip"];
	        this.isCloudflare = source["isCloudflare"];
	        this.provider = source["provider"];
	        this.serverHeader = source["serverHeader"];
	    }
	}
	export class FontDetail {
	    family: string;
	    url: string;
	    type: string;
	    transferSize: number;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new FontDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = source["family"];
	        this.url = source["url"];
	        this.type = source["type"];
	        this.transferSize = source["transferSize"];
	        this.duration = source["duration"];
	    }
	}
	export class FormFieldDetail {
	    name: string;
	    label: string;
	    type: string;
	    isRequired: boolean;
	    validationPattern?: string;
	    accept?: string;
	    value?: string;
	
	    static createFrom(source: any = {}) {
	        return new FormFieldDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.isRequired = source["isRequired"];
	        this.validationPattern = source["validationPattern"];
	        this.accept = source["accept"];
	        this.value = source["value"];
	    }
	}
	export class FormDetail {
	    id: string;
	    title: string;
	    engine: string;
	    method: string;
	    action: string;
	    fields: FormFieldDetail[];
	    fieldCount: number;
	    hasFileUpload: boolean;
	    allowedFileTypes?: string;
	    captcha: CaptchaDetail;
	    hiddenTokens?: Record<string, string>;
	    inViewport: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FormDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.engine = source["engine"];
	        this.method = source["method"];
	        this.action = source["action"];
	        this.fields = this.convertValues(source["fields"], FormFieldDetail);
	        this.fieldCount = source["fieldCount"];
	        this.hasFileUpload = source["hasFileUpload"];
	        this.allowedFileTypes = source["allowedFileTypes"];
	        this.captcha = this.convertValues(source["captcha"], CaptchaDetail);
	        this.hiddenTokens = source["hiddenTokens"];
	        this.inViewport = source["inViewport"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class IframeDetail {
	    src: string;
	    title: string;
	    width: number;
	    height: number;
	    isLazy: boolean;
	    loadedDuringScan: boolean;
	    inViewport: boolean;
	    sandbox: boolean;
	    duration: number;
	    transferSize: number;
	    formattedSize: string;
	
	    static createFrom(source: any = {}) {
	        return new IframeDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.src = source["src"];
	        this.title = source["title"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.isLazy = source["isLazy"];
	        this.loadedDuringScan = source["loadedDuringScan"];
	        this.inViewport = source["inViewport"];
	        this.sandbox = source["sandbox"];
	        this.duration = source["duration"];
	        this.transferSize = source["transferSize"];
	        this.formattedSize = source["formattedSize"];
	    }
	}
	export class ImageDetail {
	    url: string;
	    transferSize: number;
	    encodedSize: number;
	    duration: number;
	    width: number;
	    height: number;
	    naturalWidth: number;
	    naturalHeight: number;
	    renderedWidth: number;
	    renderedHeight: number;
	    formattedSize: string;
	    format: string;
	    isLazy: boolean;
	    alt: string;
	    isLCP: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImageDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.transferSize = source["transferSize"];
	        this.encodedSize = source["encodedSize"];
	        this.duration = source["duration"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.naturalWidth = source["naturalWidth"];
	        this.naturalHeight = source["naturalHeight"];
	        this.renderedWidth = source["renderedWidth"];
	        this.renderedHeight = source["renderedHeight"];
	        this.formattedSize = source["formattedSize"];
	        this.format = source["format"];
	        this.isLazy = source["isLazy"];
	        this.alt = source["alt"];
	        this.isLCP = source["isLCP"];
	    }
	}
	
	export class ResourceTiming {
	    name: string;
	    duration: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourceTiming(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.duration = source["duration"];
	        this.type = source["type"];
	    }
	}
	export class PageDiagnostics {
	    dnsTime: number;
	    tcpTime: number;
	    tlsTime: number;
	    serverProcessing: number;
	    renderBlockingCount: number;
	    renderBlockingFiles: string[];
	    lcpElement: string;
	    lcpUrl: string;
	    shiftCount: number;
	    shiftCauses: string[];
	    longTasksCount: number;
	    maxLongTaskMs: number;
	    slowestResources: ResourceTiming[];
	    largestImages: ImageDetail[];
	    fonts: FontDetail[];
	    iframes: IframeDetail[];
	    forms: FormDetail[];
	    categories: Record<string, CategoryDiagnostic>;
	    w3c?: any;
	
	    static createFrom(source: any = {}) {
	        return new PageDiagnostics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dnsTime = source["dnsTime"];
	        this.tcpTime = source["tcpTime"];
	        this.tlsTime = source["tlsTime"];
	        this.serverProcessing = source["serverProcessing"];
	        this.renderBlockingCount = source["renderBlockingCount"];
	        this.renderBlockingFiles = source["renderBlockingFiles"];
	        this.lcpElement = source["lcpElement"];
	        this.lcpUrl = source["lcpUrl"];
	        this.shiftCount = source["shiftCount"];
	        this.shiftCauses = source["shiftCauses"];
	        this.longTasksCount = source["longTasksCount"];
	        this.maxLongTaskMs = source["maxLongTaskMs"];
	        this.slowestResources = this.convertValues(source["slowestResources"], ResourceTiming);
	        this.largestImages = this.convertValues(source["largestImages"], ImageDetail);
	        this.fonts = this.convertValues(source["fonts"], FontDetail);
	        this.iframes = this.convertValues(source["iframes"], IframeDetail);
	        this.forms = this.convertValues(source["forms"], FormDetail);
	        this.categories = this.convertValues(source["categories"], CategoryDiagnostic, true);
	        this.w3c = source["w3c"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WebVitals {
	    ttfb: number;
	    fcp: number;
	    lcp: number;
	    cls: number;
	    tbt: number;
	
	    static createFrom(source: any = {}) {
	        return new WebVitals(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ttfb = source["ttfb"];
	        this.fcp = source["fcp"];
	        this.lcp = source["lcp"];
	        this.cls = source["cls"];
	        this.tbt = source["tbt"];
	    }
	}
	export class PageResult {
	    id: number;
	    url: string;
	    statusCode: number;
	    pluginCacheStatus: string;
	    cloudflareCacheStatus: string;
	    cloudflarePop?: string;
	    cloudflareRay?: string;
	    metrics: WebVitals;
	    grades: DetailedGrades;
	    diagnostics: PageDiagnostics;
	    overallStatus: string;
	    error: string;
	    recommendations: string[];
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new PageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.statusCode = source["statusCode"];
	        this.pluginCacheStatus = source["pluginCacheStatus"];
	        this.cloudflareCacheStatus = source["cloudflareCacheStatus"];
	        this.cloudflarePop = source["cloudflarePop"];
	        this.cloudflareRay = source["cloudflareRay"];
	        this.metrics = this.convertValues(source["metrics"], WebVitals);
	        this.grades = this.convertValues(source["grades"], DetailedGrades);
	        this.diagnostics = this.convertValues(source["diagnostics"], PageDiagnostics);
	        this.overallStatus = source["overallStatus"];
	        this.error = source["error"];
	        this.recommendations = source["recommendations"];
	        this.durationMs = source["durationMs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace w3c {
	
	export class W3CMessage {
	    type: string;
	    subType?: string;
	    lastLine: number;
	    lastColumn: number;
	    firstColumn?: number;
	    message: string;
	    extract?: string;
	    hiliteStart?: number;
	    hiliteLength?: number;
	
	    static createFrom(source: any = {}) {
	        return new W3CMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.subType = source["subType"];
	        this.lastLine = source["lastLine"];
	        this.lastColumn = source["lastColumn"];
	        this.firstColumn = source["firstColumn"];
	        this.message = source["message"];
	        this.extract = source["extract"];
	        this.hiliteStart = source["hiliteStart"];
	        this.hiliteLength = source["hiliteLength"];
	    }
	}
	export class W3CReport {
	    url: string;
	    errorCount: number;
	    warningCount: number;
	    isValid: boolean;
	    status: string;
	    messages: W3CMessage[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new W3CReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.errorCount = source["errorCount"];
	        this.warningCount = source["warningCount"];
	        this.isValid = source["isValid"];
	        this.status = source["status"];
	        this.messages = this.convertValues(source["messages"], W3CMessage);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace wpexport {
	
	export class ExportResult {
	    applyPHP: string;
	    rollbackPHP: string;
	    reviewZIP: string;
	    packageDir: string;
	    webpCount: number;
	    wordpressPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applyPHP = source["applyPHP"];
	        this.rollbackPHP = source["rollbackPHP"];
	        this.reviewZIP = source["reviewZIP"];
	        this.packageDir = source["packageDir"];
	        this.webpCount = source["webpCount"];
	        this.wordpressPath = source["wordpressPath"];
	    }
	}

}


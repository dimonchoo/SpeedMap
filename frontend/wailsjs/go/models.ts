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
	    globalFixes: string[];
	    totalImagePayloadBytes: number;
	    totalImagePayloadFormatted: string;
	    totalImageCount: number;
	    heavyImagesCount: number;
	    nonWebPCount: number;
	    missingLazyCount: number;
	    totalWebPSavingsBytes: number;
	    totalWebPSavingsFormatted: string;
	    formatBreakdown: Record<string, number>;
	
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
	        this.globalFixes = source["globalFixes"];
	        this.totalImagePayloadBytes = source["totalImagePayloadBytes"];
	        this.totalImagePayloadFormatted = source["totalImagePayloadFormatted"];
	        this.totalImageCount = source["totalImageCount"];
	        this.heavyImagesCount = source["heavyImagesCount"];
	        this.nonWebPCount = source["nonWebPCount"];
	        this.missingLazyCount = source["missingLazyCount"];
	        this.totalWebPSavingsBytes = source["totalWebPSavingsBytes"];
	        this.totalWebPSavingsFormatted = source["totalWebPSavingsFormatted"];
	        this.formatBreakdown = source["formatBreakdown"];
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
	    originalBytes: number;
	    originalFormatted: string;
	    optimizedBytes: number;
	    optimizedFormatted: string;
	    savingsBytes: number;
	    savingsFormatted: string;
	    savingsPercent: number;
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
	        this.originalBytes = source["originalBytes"];
	        this.originalFormatted = source["originalFormatted"];
	        this.optimizedBytes = source["optimizedBytes"];
	        this.optimizedFormatted = source["optimizedFormatted"];
	        this.savingsBytes = source["savingsBytes"];
	        this.savingsFormatted = source["savingsFormatted"];
	        this.savingsPercent = source["savingsPercent"];
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
	export class ImageDetail {
	    url: string;
	    transferSize: number;
	    encodedSize: number;
	    duration: number;
	    width: number;
	    height: number;
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


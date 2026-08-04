export namespace analytics {
	
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
	    globalFixes: string[];
	
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
	        this.globalFixes = source["globalFixes"];
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
	    authUser: string;
	    authPass: string;
	    headers: CustomHeader[];
	    isMobile: boolean;
	    timeoutSec: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapUrl = source["sitemapUrl"];
	        this.concurrency = source["concurrency"];
	        this.authUser = source["authUser"];
	        this.authPass = source["authPass"];
	        this.headers = this.convertValues(source["headers"], CustomHeader);
	        this.isMobile = source["isMobile"];
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


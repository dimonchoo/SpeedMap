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


export namespace main {
	
	export class AppSettings {
	    auto_change_enabled: boolean;
	    change_interval_hours: number;
	    download_sources: string[];
	    max_wallpapers: number;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_change_enabled = source["auto_change_enabled"];
	        this.change_interval_hours = source["change_interval_hours"];
	        this.download_sources = source["download_sources"];
	        this.max_wallpapers = source["max_wallpapers"];
	    }
	}
	export class WallpaperInfo {
	    id: string;
	    filename: string;
	    filepath: string;
	    local_url: string;
	    // Go type: time
	    download_date: any;
	    source_url: string;
	    file_size: number;
	
	    static createFrom(source: any = {}) {
	        return new WallpaperInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.filepath = source["filepath"];
	        this.local_url = source["local_url"];
	        this.download_date = this.convertValues(source["download_date"], null);
	        this.source_url = source["source_url"];
	        this.file_size = source["file_size"];
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


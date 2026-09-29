export namespace main {
	
	export class Audit {
	    id: number;
	    // Go type: time
	    ts: any;
	    serverId?: number;
	    action: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new Audit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ts = this.convertValues(source["ts"], null);
	        this.serverId = source["serverId"];
	        this.action = source["action"];
	        this.detail = source["detail"];
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
	export class FileContent {
	    path: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new FileContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	    }
	}
	export class FileInfo {
	    name: string;
	    path: string;
	    size: number;
	    dir: boolean;
	    mode: string;
	    modTime: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.dir = source["dir"];
	        this.mode = source["mode"];
	        this.modTime = source["modTime"];
	    }
	}
	export class FileListResult {
	    root: string;
	    files: FileInfo[];
	
	    static createFrom(source: any = {}) {
	        return new FileListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.files = this.convertValues(source["files"], FileInfo);
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
	export class FileOpInput {
	    projectId: number;
	    path: string;
	    dir: string;
	    name: string;
	    confirm: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileOpInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.path = source["path"];
	        this.dir = source["dir"];
	        this.name = source["name"];
	        this.confirm = source["confirm"];
	    }
	}
	export class FileWriteInput {
	    projectId: number;
	    path: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new FileWriteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.path = source["path"];
	        this.content = source["content"];
	    }
	}
	export class Meta {
	    dataDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Meta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataDir = source["dataDir"];
	    }
	}
	export class Project {
	    id: number;
	    serverId: number;
	    name: string;
	    remotePath: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.name = source["name"];
	        this.remotePath = source["remotePath"];
	    }
	}
	export class ProjectInput {
	    serverId: number;
	    name: string;
	    remotePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.name = source["name"];
	        this.remotePath = source["remotePath"];
	    }
	}
	export class Server {
	    id: number;
	    name: string;
	    host: string;
	    port: number;
	    username: string;
	    authType: string;
	    hasPassword: boolean;
	    hasKey: boolean;
	    hasHostKey: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.authType = source["authType"];
	        this.hasPassword = source["hasPassword"];
	        this.hasKey = source["hasKey"];
	        this.hasHostKey = source["hasHostKey"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ServerInput {
	    name: string;
	    host: string;
	    port: number;
	    username: string;
	    authType: string;
	    password?: string;
	    privateKey?: string;
	    passphrase?: string;
	    hostKey: string;
	    clearHostKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.authType = source["authType"];
	        this.password = source["password"];
	        this.privateKey = source["privateKey"];
	        this.passphrase = source["passphrase"];
	        this.hostKey = source["hostKey"];
	        this.clearHostKey = source["clearHostKey"];
	    }
	}
	export class TestInput {
	    serverId: number;
	    host: string;
	    port: number;
	    username: string;
	    authType: string;
	    password?: string;
	    privateKey?: string;
	    passphrase?: string;
	    acceptHostKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TestInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.authType = source["authType"];
	        this.password = source["password"];
	        this.privateKey = source["privateKey"];
	        this.passphrase = source["passphrase"];
	        this.acceptHostKey = source["acceptHostKey"];
	    }
	}
	export class TestResult {
	    ok: boolean;
	    fingerprint: string;
	    hostKey: string;
	    error: string;
	    needHostKey: boolean;
	    hostChanged: boolean;
	    fingerprintUI: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.fingerprint = source["fingerprint"];
	        this.hostKey = source["hostKey"];
	        this.error = source["error"];
	        this.needHostKey = source["needHostKey"];
	        this.hostChanged = source["hostChanged"];
	        this.fingerprintUI = source["fingerprintUI"];
	    }
	}

}


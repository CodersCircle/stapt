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
	export class CommandResult {
	    ok: boolean;
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class Project {
	    id: number;
	    serverId: number;
	    name: string;
	    remotePath: string;
	    workPath: string;
	    kind: string;
	    status: string;
	    lastChecked: string;
	    tech: string;
	    lastUploadAt: string;
	    lastUploadFile: string;
	    lastUploadSize: number;
	    lastUploadStatus: string;
	    pathTestStatus: string;
	    pathTestUrl: string;
	    pathTestAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.name = source["name"];
	        this.remotePath = source["remotePath"];
	        this.workPath = source["workPath"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.lastChecked = source["lastChecked"];
	        this.tech = source["tech"];
	        this.lastUploadAt = source["lastUploadAt"];
	        this.lastUploadFile = source["lastUploadFile"];
	        this.lastUploadSize = source["lastUploadSize"];
	        this.lastUploadStatus = source["lastUploadStatus"];
	        this.pathTestStatus = source["pathTestStatus"];
	        this.pathTestUrl = source["pathTestUrl"];
	        this.pathTestAt = source["pathTestAt"];
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
	export class ConnectDiscoverResult {
	    test: TestResult;
	    projects: Project[];
	
	    static createFrom(source: any = {}) {
	        return new ConnectDiscoverResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.test = this.convertValues(source["test"], TestResult);
	        this.projects = this.convertValues(source["projects"], Project);
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
	export class Note {
	    id: number;
	    projectId: number;
	    category: string;
	    title: string;
	    content: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class NoteInput {
	    projectId: number;
	    category: string;
	    title: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new NoteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.content = source["content"];
	    }
	}
	export class ParsedSSH {
	    host: string;
	    port: number;
	    username: string;
	
	    static createFrom(source: any = {}) {
	        return new ParsedSSH(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	    }
	}
	export class PathTestResult {
	    ok: boolean;
	    dest: string;
	    url: string;
	    error: string;
	    marker: string;
	
	    static createFrom(source: any = {}) {
	        return new PathTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.dest = source["dest"];
	        this.url = source["url"];
	        this.error = source["error"];
	        this.marker = source["marker"];
	    }
	}
	
	export class ProjectInput {
	    serverId: number;
	    name: string;
	    remotePath: string;
	    kind: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.name = source["name"];
	        this.remotePath = source["remotePath"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	    }
	}
	export class QuickCommand {
	    key: string;
	    group: string;
	    label: string;
	    destructive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QuickCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.group = source["group"];
	        this.label = source["label"];
	        this.destructive = source["destructive"];
	    }
	}
	export class SavePathInput {
	    projectId: number;
	    name: string;
	    dir: string;
	
	    static createFrom(source: any = {}) {
	        return new SavePathInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.name = source["name"];
	        this.dir = source["dir"];
	    }
	}
	export class SavedPath {
	    id: number;
	    serverId: number;
	    projectId: number;
	    name: string;
	    remotePath: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedPath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.projectId = source["projectId"];
	        this.name = source["name"];
	        this.remotePath = source["remotePath"];
	        this.createdAt = source["createdAt"];
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
	    lastConnected: string;
	    projectCount: number;
	    pathCount: number;
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
	        this.lastConnected = source["lastConnected"];
	        this.projectCount = source["projectCount"];
	        this.pathCount = source["pathCount"];
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
	
	export class UploadItem {
	    rel: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new UploadItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rel = source["rel"];
	        this.size = source["size"];
	    }
	}
	export class UploadPreview {
	    kind: string;
	    dest: string;
	    files: number;
	    folders: number;
	    bytes: number;
	    items: UploadItem[];
	    skipped: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.dest = source["dest"];
	        this.files = source["files"];
	        this.folders = source["folders"];
	        this.bytes = source["bytes"];
	        this.items = this.convertValues(source["items"], UploadItem);
	        this.skipped = source["skipped"];
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
	export class UploadResult {
	    ok: boolean;
	    done: number;
	    failed: number;
	    errors: string[];
	    skipped: boolean;
	    dest: string;
	    label: string;
	    kind: string;
	    bytes: number;
	    time: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.done = source["done"];
	        this.failed = source["failed"];
	        this.errors = source["errors"];
	        this.skipped = source["skipped"];
	        this.dest = source["dest"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.bytes = source["bytes"];
	        this.time = source["time"];
	    }
	}

}


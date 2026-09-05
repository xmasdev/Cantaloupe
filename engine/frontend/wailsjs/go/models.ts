export namespace main {

	export class DownloadStatus {
	    state: string;
	    name: string;
	    torrentPath: string;
	    outputDir: string;
	    totalBytes: number;
	    downloadedBytes: number;
	    pieceCount: number;
	    completedPieces: number;
	    peerCount: number;
	    port: number;
	    error: string;
	    downloadSpeed: number;
	    uploadSpeed: number;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new DownloadStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.name = source["name"];
	        this.torrentPath = source["torrentPath"];
	        this.outputDir = source["outputDir"];
	        this.totalBytes = source["totalBytes"];
	        this.downloadedBytes = source["downloadedBytes"];
	        this.pieceCount = source["pieceCount"];
	        this.completedPieces = source["completedPieces"];
	        this.peerCount = source["peerCount"];
	        this.port = source["port"];
	        this.error = source["error"];
	        this.downloadSpeed = source["downloadSpeed"];
	        this.uploadSpeed = source["uploadSpeed"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class TorrentFile {
	    path: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new TorrentFile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	    }
	}
	export class TorrentPeer {
	    address: string;
	    pieces: number;
	    state: string;

	    static createFrom(source: any = {}) {
	        return new TorrentPeer(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.address = source["address"];
	        this.pieces = source["pieces"];
	        this.state = source["state"];
	    }
	}
	export class TorrentTracker {
	    url: string;
	    status: string;

	    static createFrom(source: any = {}) {
	        return new TorrentTracker(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.status = source["status"];
	    }
	}
}

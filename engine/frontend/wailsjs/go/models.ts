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
	        this.updatedAt = source["updatedAt"];
	    }
	}

}


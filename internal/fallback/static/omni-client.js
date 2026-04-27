class OmniClient {
    constructor(options = {}) {
        this.workerPath = options.workerPath || '/omni-worker.js';
        this.maxWorkers = options.maxWorkers || (navigator.hardwareConcurrency > 4 ? 8 : 4);
        this.chunkSize = options.chunkSize || 1024 * 1024; // 1MB
        this.secret = options.secret || "default-secret";
        this.workers = [];
        this.workerQueue = [];
        this.activeTasks = 0;
        this.taskIdCounter = 0;

        this.initWorkers();
    }

    initWorkers() {
        for (let i = 0; i < this.maxWorkers; i++) {
            const worker = new Worker(this.workerPath);
            worker.onmessage = this.onWorkerMessage.bind(this);
            this.workers.push({ worker, busy: false, currentTask: null });
        }
    }

    onWorkerMessage(e) {
        const { type, payload } = e.data;
        const workerEntry = this.workers.find(w => w.currentTask && w.currentTask.id === payload.taskId);

        if (workerEntry) {
            const task = workerEntry.currentTask;
            workerEntry.busy = false;
            workerEntry.currentTask = null;
            this.activeTasks--;

            if (type === 'chunk_success') {
                task.resolve(payload.data);
            } else {
                task.reject(new Error(payload.error));
            }

            this.processQueue();
        }
    }

    processQueue() {
        if (this.workerQueue.length === 0) return;

        const availableWorker = this.workers.find(w => !w.busy);
        if (availableWorker) {
            const task = this.workerQueue.shift();
            availableWorker.busy = true;
            availableWorker.currentTask = task;

            // Pass the taskId into the worker so it can return it
            task.config.taskId = task.id;

            availableWorker.worker.postMessage({ type: 'fetch', payload: task.config });
            this.activeTasks++;
        }
    }

    scheduleChunk(config) {
        return new Promise((resolve, reject) => {
            this.taskIdCounter++;
            const task = { id: this.taskIdCounter, config, resolve, reject, offset: config.offset };
            this.workerQueue.push(task);
            this.processQueue();
        });
    }

    async fetchMedia(url, fileId, totalSize, isVideo = true) {
        const chunks = [];
        const promises = [];

        for (let offset = 0; offset < totalSize; offset += this.chunkSize) {
            const length = Math.min(this.chunkSize, totalSize - offset);
            promises.push(this.scheduleChunk({
                url,
                fileId,
                offset,
                length,
                secret: this.secret,
                useWebTransport: false
            }).then(data => ({ offset, data, length })));
        }

        if (!isVideo) {
            const results = await Promise.all(promises);
            results.sort((a, b) => a.offset - b.offset);

            let totalLength = results.reduce((acc, r) => acc + r.data.byteLength, 0);
            const combined = new Uint8Array(totalLength);
            let pos = 0;
            for (const r of results) {
                combined.set(new Uint8Array(r.data), pos);
                pos += r.data.byteLength;
            }
            return new Blob([combined]);
        } else {
            return promises;
        }
    }

    async renderVideo(videoElement, url, fileId, totalSize, mimeCodec = 'video/mp4; codecs="avc1.42E01E, mp4a.40.2"') {
        if (!window.MediaSource) {
            throw new Error('MediaSource API not supported');
        }

        const mediaSource = new MediaSource();
        videoElement.src = URL.createObjectURL(mediaSource);

        await new Promise(resolve => mediaSource.addEventListener('sourceopen', resolve, { once: true }));

        const sourceBuffer = mediaSource.addSourceBuffer(mimeCodec);
        sourceBuffer.mode = 'segments';

        let pendingChunks = new Map();
        let nextExpectedOffset = 0;
        let isAppending = false;

        const processAppends = async () => {
            if (isAppending) return;

            if (pendingChunks.has(nextExpectedOffset)) {
                isAppending = true;
                const chunkInfo = pendingChunks.get(nextExpectedOffset);
                pendingChunks.delete(nextExpectedOffset);

                sourceBuffer.appendBuffer(chunkInfo.data);
                await new Promise(resolve => sourceBuffer.addEventListener('updateend', resolve, { once: true }));

                nextExpectedOffset += chunkInfo.length;
                isAppending = false;

                // Immediately try to process the next sequential piece
                processAppends();
            }
        };

        const chunkPromises = await this.fetchMedia(url, fileId, totalSize, true);

        let chunksProcessed = 0;
        const onChunkFinished = () => {
            chunksProcessed++;
            if (chunksProcessed === chunkPromises.length) {
                const checkFinished = setInterval(() => {
                    if (pendingChunks.size === 0 && !isAppending && mediaSource.readyState === 'open') {
                        mediaSource.endOfStream();
                        clearInterval(checkFinished);
                    }
                }, 100);
            }
        };

        chunkPromises.forEach(p => p.then((chunkInfo) => {
            pendingChunks.set(chunkInfo.offset, chunkInfo);
            processAppends();
            onChunkFinished();
        }).catch(err => {
            console.error("Chunk fetch error", err);
            onChunkFinished();
        }));
    }
}
window.OmniClient = OmniClient;

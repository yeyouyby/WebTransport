const MAGIC = 0x4D4D4746;
const HEADER_SIZE = 32;

function buildHeader(fileId, offset, length, secret = new Uint8Array(4)) {
    const buffer = new ArrayBuffer(HEADER_SIZE);
    const view = new DataView(buffer);

    view.setUint32(0, MAGIC, false);
    view.setUint32(4, Math.floor(Date.now() / 1000), false);
    view.setUint32(8, 0, false);
    view.setUint32(12, Math.floor(Math.random() * 0xffffffff), false);
    view.setUint32(16, Math.floor(offset / 0x100000000), false);
    view.setUint32(20, offset % 0x100000000, false);
    view.setUint32(24, length, false);
    view.setUint32(28, 0, false);

    return buffer;
}

async function fetchUsingFetch(url, headerBuffer) {
    const response = await fetch(url, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/octet-stream'
        },
        body: headerBuffer
    });

    if (!response.ok) {
        throw new Error(`Fetch failed with status: ${response.status}`);
    }

    const arrayBuffer = await response.arrayBuffer();

    if (arrayBuffer.byteLength < HEADER_SIZE) {
        throw new Error('Response too short');
    }

    const payload = arrayBuffer.slice(HEADER_SIZE);
    return payload;
}

async function fetchChunk(config) {
    const { url, fileId, offset, length, useWebTransport } = config;
    const header = buildHeader(fileId, offset, length);
    return await fetchUsingFetch(url, header);
}

self.onmessage = async (e) => {
    const { type, payload } = e.data;
    if (type === 'fetch') {
        try {
            const dataBuffer = await fetchChunk(payload);
            self.postMessage({
                type: 'chunk_success',
                payload: {
                    offset: payload.offset,
                    data: dataBuffer
                }
            }, [dataBuffer]);
        } catch (error) {
            self.postMessage({
                type: 'chunk_error',
                payload: {
                    offset: payload.offset,
                    error: error.message
                }
            });
        }
    }
};

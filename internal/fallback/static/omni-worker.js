const MAGIC = 0x4D4D4746;
const HEADER_SIZE = 32;

async function buildHeader(fileId, offset, length, secretString = "default-secret") {
    const buffer = new ArrayBuffer(HEADER_SIZE);
    const view = new DataView(buffer);

    view.setUint32(0, MAGIC, false);
    view.setUint32(4, Math.floor(Date.now() / 1000), false);
    view.setUint32(8, 0, false);
    view.setUint32(12, Math.floor(Math.random() * 0xffffffff), false);
    view.setUint32(16, Math.floor(offset / 0x100000000), false);
    view.setUint32(20, offset % 0x100000000, false);
    view.setUint32(24, length, false);
    // Token placeholder
    view.setUint32(28, 0, false);

    // Compute HMAC-SHA256 for the first 28 bytes
    const encoder = new TextEncoder();
    const keyData = encoder.encode(secretString);

    let cryptoKey;
    try {
        cryptoKey = await crypto.subtle.importKey(
            "raw",
            keyData,
            { name: "HMAC", hash: "SHA-256" },
            false,
            ["sign"]
        );

        const headerWithoutToken = buffer.slice(0, 28);
        const signature = await crypto.subtle.sign("HMAC", cryptoKey, headerWithoutToken);
        const sigView = new DataView(signature);

        // Write the first 4 bytes of the HMAC as the token
        view.setUint32(28, sigView.getUint32(0, false), false);
    } catch (e) {
        console.warn("Failed to sign header, using token 0", e);
    }

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
    const { url, fileId, offset, length, useWebTransport, secret } = config;
    const header = await buildHeader(fileId, offset, length, secret);
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
                    taskId: payload.taskId,
                    offset: payload.offset,
                    data: dataBuffer
                }
            }, [dataBuffer]);
        } catch (error) {
            self.postMessage({
                type: 'chunk_error',
                payload: {
                    taskId: payload.taskId,
                    offset: payload.offset,
                    error: error.message
                }
            });
        }
    }
};

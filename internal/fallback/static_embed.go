package fallback

import _ "embed"

//go:embed static/shard-client.js
var shardClientScript []byte

//go:embed static/omni-client.js
var omniClientScript []byte

//go:embed static/omni-worker.js
var omniWorkerScript []byte

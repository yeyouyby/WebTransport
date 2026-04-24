package fallback

import _ "embed"

//go:embed static/shard-client.js
var shardClientScript []byte

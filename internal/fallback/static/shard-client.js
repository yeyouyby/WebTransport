(function () {
  "use strict";

  var IMAGE_FORMATS = ["jpg", "jpeg", "png", "webp", "avif", "gif", "bmp", "svg"];
  var VIDEO_FORMATS = ["mp4", "m4v", "webm", "mkv", "mov", "ts", "m3u8", "flv"];
  var AUDIO_FORMATS = ["mp3", "aac", "m4a", "flac", "ogg", "opus", "wav", "amr"];
  var MIME_HINTS = {
    "image/jpeg": "image",
    "image/png": "image",
    "image/webp": "image",
    "image/avif": "image",
    "image/gif": "image",
    "video/mp4": "video",
    "video/webm": "video",
    "video/quicktime": "video",
    "application/vnd.apple.mpegurl": "video",
    "audio/mpeg": "audio",
    "audio/aac": "audio",
    "audio/flac": "audio",
    "audio/ogg": "audio",
    "audio/wav": "audio"
  };

  function utf8Bytes(input) {
    if (typeof TextEncoder !== "undefined") {
      return new TextEncoder().encode(input);
    }
    var out = [];
    for (var i = 0; i < input.length; i++) {
      var c = input.charCodeAt(i);
      if (c < 0x80) {
        out.push(c);
      } else if (c < 0x800) {
        out.push(0xc0 | (c >> 6), 0x80 | (c & 0x3f));
      } else if (c < 0xd800 || c >= 0xe000) {
        out.push(0xe0 | (c >> 12), 0x80 | ((c >> 6) & 0x3f), 0x80 | (c & 0x3f));
      } else {
        i += 1;
        if (i >= input.length) {
          out.push(0xef, 0xbf, 0xbd);
          break;
        }
        var c2 = input.charCodeAt(i);
        if (c2 < 0xdc00 || c2 > 0xdfff) {
          out.push(0xef, 0xbf, 0xbd);
          i -= 1;
          continue;
        }
        var codePoint = 0x10000 + (((c & 0x3ff) << 10) | (c2 & 0x3ff));
        out.push(
          0xf0 | (codePoint >> 18),
          0x80 | ((codePoint >> 12) & 0x3f),
          0x80 | ((codePoint >> 6) & 0x3f),
          0x80 | (codePoint & 0x3f)
        );
      }
    }
    return out;
  }

  function fnv1a64(input) {
    var hash = 0xcbf29ce484222325n;
    var prime = 0x100000001b3n;
    var bytes = utf8Bytes(input);
    for (var i = 0; i < bytes.length; i++) {
      hash ^= BigInt(bytes[i]);
      hash = BigInt.asUintN(64, hash * prime);
    }
    return hash;
  }

  function pickHost(hosts, key) {
    if (!hosts || hosts.length === 0) {
      throw new Error("shard hosts is empty");
    }
    if (!key) {
      return hosts[0];
    }
    var best = hosts[0];
    var bestScore = -1n;
    for (var i = 0; i < hosts.length; i++) {
      var score = fnv1a64(hosts[i] + ":" + key);
      if (score > bestScore) {
        bestScore = score;
        best = hosts[i];
      }
    }
    return best;
  }

  function normalizeExt(input) {
    if (!input) {
      return "";
    }
    var clean = String(input).toLowerCase().trim();
    if (clean.indexOf(".") >= 0) {
      clean = clean.split(".").pop();
    }
    if (clean.indexOf("/") >= 0) {
      clean = clean.split("/").pop();
    }
    if (clean.indexOf(";") >= 0) {
      clean = clean.split(";")[0];
    }
    return clean;
  }

  function mergeUint8Arrays(parts, totalLength) {
    var out = new Uint8Array(totalLength);
    var cursor = 0;
    for (var i = 0; i < parts.length; i++) {
      out.set(parts[i], cursor);
      cursor += parts[i].length;
    }
    return out;
  }

  function toQuery(params) {
    var parts = [];
    Object.keys(params || {}).forEach(function (k) {
      if (params[k] === undefined || params[k] === null) {
        return;
      }
      parts.push(encodeURIComponent(k) + "=" + encodeURIComponent(String(params[k])));
    });
    return parts.length ? "?" + parts.join("&") : "";
  }

  function MatrixShardClient(opts) {
    opts = opts || {};
    this.scheme = opts.scheme || "https";
    this.imageShards = opts.imageShards || [];
    this.videoShards = opts.videoShards || [];
    this.audioShards = opts.audioShards || [];
    this.defaultPath = opts.defaultPath || "/fallback";
    this.maxConcurrency = typeof opts.maxConcurrency === "number" ? opts.maxConcurrency : 4;
    this.defaultChunkSize = typeof opts.defaultChunkSize === "number" ? opts.defaultChunkSize : 256 * 1024;
  }

  MatrixShardClient.prototype.pick = function (resourceType, key) {
    var hosts;
    if (resourceType === "image") {
      hosts = this.imageShards;
    } else if (resourceType === "audio") {
      hosts = this.audioShards;
    } else {
      hosts = this.videoShards;
    }
    return pickHost(hosts, key);
  };

  MatrixShardClient.prototype.detectResourceType = function (formatOrMime, fallbackType) {
    var raw = String(formatOrMime || "").toLowerCase().trim();
    if (MIME_HINTS[raw]) {
      return MIME_HINTS[raw];
    }

    var ext = normalizeExt(raw);
    if (IMAGE_FORMATS.indexOf(ext) >= 0) {
      return "image";
    }
    if (VIDEO_FORMATS.indexOf(ext) >= 0) {
      return "video";
    }
    if (AUDIO_FORMATS.indexOf(ext) >= 0) {
      return "audio";
    }
    return fallbackType || "video";
  };

  MatrixShardClient.prototype.buildURL = function (resourceType, path, query, key) {
    var host = this.pick(resourceType, key);
    return this.scheme + "://" + host + (path || this.defaultPath) + toQuery(query);
  };

  MatrixShardClient.prototype.fetchRange = function (resourceType, key, offset, length, extraHeaders, path, signal) {
    var url = this.buildURL(resourceType, path || this.defaultPath, null, key);
    var headers = Object.assign({}, extraHeaders || {});
    headers.Range = "bytes=" + offset + "-" + (offset + length - 1);
    return fetch(url, {
      method: "GET",
      headers: headers,
      credentials: "include",
      signal: signal
    });
  };

  MatrixShardClient.prototype.fetchSegmented = async function (opts) {
    opts = opts || {};
    var resourceType = opts.resourceType || this.detectResourceType(opts.format, "video");
    var key = opts.key || "";
    var offset = opts.offset || 0;
    var length = opts.length || 0;
    var chunkSize = opts.chunkSize || this.defaultChunkSize;
    var concurrency = opts.concurrency || this.maxConcurrency;
    var headers = opts.headers || {};

    if (!length || length <= 0) {
      throw new Error("length must be greater than 0");
    }
    if (chunkSize <= 0) {
      chunkSize = this.defaultChunkSize;
    }
    if (concurrency <= 0) {
      concurrency = 1;
    }

    var segments = [];
    var remain = length;
    var cursor = offset;
    while (remain > 0) {
      var current = Math.min(remain, chunkSize);
      segments.push({ offset: cursor, length: current });
      cursor += current;
      remain -= current;
    }

    var outParts = new Array(segments.length);
    var nextIndex = 0;

    var runWorker = async () => {
      while (true) {
        var idx = nextIndex;
        nextIndex += 1;
        if (idx >= segments.length) {
          return;
        }
        var seg = segments[idx];
        var resp = await this.fetchRange(resourceType, key, seg.offset, seg.length, headers, opts.path, opts.signal);
        if (!resp.ok && resp.status !== 206) {
          throw new Error("segment fetch failed: " + resp.status);
        }
        var buf = await resp.arrayBuffer();
        outParts[idx] = new Uint8Array(buf);
      }
    };

    var workers = [];
    var workerCount = Math.min(concurrency, segments.length);
    for (var i = 0; i < workerCount; i++) {
      workers.push(runWorker());
    }
    await Promise.all(workers);

    var total = 0;
    for (var j = 0; j < outParts.length; j++) {
      total += outParts[j].length;
    }
    return mergeUint8Arrays(outParts, total);
  };

  MatrixShardClient.prototype.fetchMedia = function (opts) {
    opts = opts || {};
    var type = opts.resourceType || this.detectResourceType(opts.format, "video");
    var useSegment = opts.segmented !== false;
    if (!useSegment) {
      return this.fetchRange(type, opts.key || "", opts.offset || 0, opts.length || 0, opts.headers, opts.path, opts.signal)
        .then(function (resp) {
          if (!resp.ok && resp.status !== 206) {
            throw new Error("fetch failed: " + resp.status);
          }
          return resp.arrayBuffer();
        })
        .then(function (buf) {
          return new Uint8Array(buf);
        });
    }
    return this.fetchSegmented(Object.assign({}, opts, { resourceType: type }));
  };

  MatrixShardClient.prototype.listCapabilities = function () {
    return {
      imageFormats: IMAGE_FORMATS.slice(),
      videoFormats: VIDEO_FORMATS.slice(),
      audioFormats: AUDIO_FORMATS.slice(),
      maxConcurrency: this.maxConcurrency,
      defaultChunkSize: this.defaultChunkSize
    };
  };

  window.MatrixShardClient = MatrixShardClient;
})();

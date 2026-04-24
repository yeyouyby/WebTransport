param(
  [ValidateSet("fallback", "bench")]
  [string]$Mode = "fallback",
  [string]$Url = "https://127.0.0.1:8443/fallback",
  [int]$Requests = 1000,
  [int]$Concurrency = 20,
  [int]$RangeSize = 262144,
  [int]$TimeoutSec = 8,
  [string]$BenchClient = "./bin/bench-client.exe",
  [string]$Endpoint = "https://127.0.0.1:8444/wt",
  [int]$Duration = 60,
  [string]$OutDir = "ops/external/results/$(Get-Date -Format 'yyyyMMdd-HHmmss')"
)

$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

if ($Mode -eq "bench") {
  if (-not (Test-Path $BenchClient)) {
    throw "bench client not found: $BenchClient"
  }
  & $BenchClient --mode datagram --endpoint $Endpoint --seconds $Duration --concurrency $Concurrency | Tee-Object -FilePath (Join-Path $OutDir "bench.log")
  Write-Host "bench mode completed: $(Join-Path $OutDir 'bench.log')"
  exit 0
}

$statusFile = Join-Path $OutDir "fallback-http-status.log"
$summaryFile = Join-Path $OutDir "fallback-summary.env"

"timestamp=$(Get-Date -Format o)" | Set-Content -Path $summaryFile -Encoding UTF8
"mode=$Mode" | Add-Content -Path $summaryFile -Encoding UTF8
"url=$Url" | Add-Content -Path $summaryFile -Encoding UTF8
"requests=$Requests" | Add-Content -Path $summaryFile -Encoding UTF8
"concurrency=$Concurrency" | Add-Content -Path $summaryFile -Encoding UTF8
"range_size=$RangeSize" | Add-Content -Path $summaryFile -Encoding UTF8

$start = Get-Date
$codes = New-Object System.Collections.Generic.List[string]

if ($PSVersionTable.PSVersion.Major -ge 7) {
  [System.Collections.Generic.List[int]]$indices = 1..$Requests
  $codes = $indices | ForEach-Object -Parallel {
    $i = $_
    $size = $using:RangeSize
    $offset = ($i - 1) * $size
    $end = $offset + $size - 1
    $headers = @{ Range = "bytes=$offset-$end" }
    try {
      $resp = Invoke-WebRequest -Uri $using:Url -Method Get -Headers $headers -TimeoutSec $using:TimeoutSec -SkipCertificateCheck
      [string]$resp.StatusCode
    }
    catch {
      if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
        [string][int]$_.Exception.Response.StatusCode
      }
      else {
        "000"
      }
    }
  } -ThrottleLimit $Concurrency
}
else {
  Add-Type @"
using System.Net;
using System.Security.Cryptography.X509Certificates;
public static class TrustAllCertsPolicy {
    public static bool IgnoreValidation(object sender, X509Certificate cert, X509Chain chain, System.Net.Security.SslPolicyErrors errors) { return true; }
}
"@
  [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { param($sender, $cert, $chain, $errors) return $true }

  $jobs = @()
  for ($i = 1; $i -le $Requests; $i++) {
    $jobs += Start-Job -ScriptBlock {
      param($idx, $u, $s, $t)
      $offset = ($idx - 1) * $s
      $end = $offset + $s - 1
      $headers = @{ Range = "bytes=$offset-$end" }
      try {
        $resp = Invoke-WebRequest -Uri $u -Method Get -Headers $headers -TimeoutSec $t
        [string]$resp.StatusCode
      }
      catch {
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
          [string][int]$_.Exception.Response.StatusCode
        }
        else {
          "000"
        }
      }
    } -ArgumentList $i, $Url, $RangeSize, $TimeoutSec

    if ($jobs.Count -ge $Concurrency) {
      Wait-Job -Job $jobs | Out-Null
      $codes.AddRange((Receive-Job -Job $jobs))
      $jobs = @()
    }
  }

  if ($jobs.Count -gt 0) {
    Wait-Job -Job $jobs | Out-Null
    $codes.AddRange((Receive-Job -Job $jobs))
  }
}

$codes | Set-Content -Path $statusFile -Encoding UTF8

$elapsedMs = [int]((Get-Date) - $start).TotalMilliseconds
if ($elapsedMs -le 0) { $elapsedMs = 1 }

$totalCount = $codes.Count
$okCount = ($codes | Where-Object { $_ -eq "200" -or $_ -eq "206" }).Count
$failCount = $totalCount - $okCount
$totalBytes = [double]($totalCount * $RangeSize)
$throughputMiB = (($totalBytes * 1000.0 / $elapsedMs) / 1MB)

"elapsed_ms=$elapsedMs" | Add-Content -Path $summaryFile -Encoding UTF8
"ok_count=$okCount" | Add-Content -Path $summaryFile -Encoding UTF8
"fail_count=$failCount" | Add-Content -Path $summaryFile -Encoding UTF8
"throughput_mib_per_sec=$('{0:N2}' -f $throughputMiB)" | Add-Content -Path $summaryFile -Encoding UTF8

Write-Host "pressure test completed"
Write-Host "  status_file: $statusFile"
Write-Host "  summary:     $summaryFile"
Write-Host "  ok/fail:     $okCount/$failCount"
Write-Host "  throughput:  $('{0:N2}' -f $throughputMiB) MiB/s"

[CmdletBinding()]
param(
    [int]$Warmup = 3,
    [int]$Samples = 7,
    [int]$MinDurationMs = 200,
    [long]$MaxIterations = 16777216,
    [int]$ProcessTimeoutSeconds = 300
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$temporaryDirectory = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("micromustache-benchmark-verify-" + [guid]::NewGuid().ToString('N'))))
if (-not $temporaryDirectory.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) { throw 'Temporary directory escaped system temp' }
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null

try {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repositoryRoot 'scripts\verify-original-tests.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'Original-test verification failed' }
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repositoryRoot 'scripts\verify-node-oracle.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'Node oracle verification failed' }
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repositoryRoot 'scripts\verify-differential.ps1') -EvidenceJson (Join-Path $temporaryDirectory 'differential.json') -EvidenceMarkdown (Join-Path $temporaryDirectory 'differential.md')
    if ($LASTEXITCODE -ne 0) { throw 'Differential regression failed' }

    $jsonReport = Join-Path $repositoryRoot 'evidence\benchmark-summary.json'
    $markdownReport = Join-Path $repositoryRoot 'evidence\benchmark-summary.md'
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repositoryRoot 'scripts\run-benchmark.ps1') -JsonReport $jsonReport -MarkdownReport $markdownReport -Warmup $Warmup -Samples $Samples -MinDurationMs $MinDurationMs -MaxIterations $MaxIterations -ProcessTimeoutSeconds $ProcessTimeoutSeconds
    if ($LASTEXITCODE -ne 0) { throw 'Benchmark run failed' }

    $report = Get-Content -Raw -Encoding UTF8 -LiteralPath $jsonReport | ConvertFrom-Json
    if ($report.schemaVersion -ne 1 -or $report.correctnessStatus -cne 'PASS') { throw 'Invalid benchmark report schema or correctness status' }
    if ($report.workloads.Count -lt 11) { throw 'Benchmark report has too few workloads' }
    $requiredAPIs = @('tokenize','get','getRef','render','renderFn','renderFnAsync','compile','renderer.construct','renderer.render','renderer.renderFn','renderer.renderFnAsync')
    foreach ($api in $requiredAPIs) { if ($report.apiCounts.PSObject.Properties.Name -notcontains $api) { throw "Missing required API: $api" } }
    foreach ($workload in $report.workloads) {
        if ($workload.correctness.status -cne 'PASS') { throw "Correctness failed: $($workload.id)" }
        if ($workload.rawSamples.Count -ne $Samples * 4) { throw "Raw sample count mismatch: $($workload.id)" }
        if ($workload.node.rounds -ne 2 -or $workload.go.rounds -ne 2 -or $workload.node.samples -ne $Samples * 2 -or $workload.go.samples -ne $Samples * 2) { throw "Round/sample aggregation mismatch: $($workload.id)" }
        if ($workload.node.medianNsPerOp -le 0 -or $workload.go.medianNsPerOp -le 0 -or $workload.goMedianOverNodeMedian -le 0) { throw "Invalid aggregate metric: $($workload.id)" }
    }
    $privatePattern = '(?i)([A-Z]:\\|/Users/|\\Users\\|hostname|user(name)?\s*[=:]|serial|MAC\s*[=:])'
    if ((Get-Content -Raw -Encoding UTF8 -LiteralPath $jsonReport) -match $privatePattern -or (Get-Content -Raw -Encoding UTF8 -LiteralPath $markdownReport) -match $privatePattern) { throw 'Private or absolute path content found in evidence' }
    Write-Output "PASS benchmark verification workloads=$($report.workloads.Count) rawSamples=$((@($report.workloads | ForEach-Object {$_.rawSamples.Count}) | Measure-Object -Sum).Sum) correctness=$($report.correctnessStatus) hash=$($report.contentSha256)"
}
finally {
    if (Test-Path -LiteralPath $temporaryDirectory) {
        $resolved = [System.IO.Path]::GetFullPath($temporaryDirectory)
        if (-not $resolved.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) { throw "Refusing to remove non-temp directory: $resolved" }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

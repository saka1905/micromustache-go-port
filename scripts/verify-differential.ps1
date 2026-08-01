[CmdletBinding()]
param(
    [int]$TimeoutSeconds = 30,
    [string]$EvidenceJson,
    [string]$EvidenceMarkdown
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$runner = Join-Path $repositoryRoot 'scripts\run-differential.ps1'
$corpus = Join-Path $repositoryRoot 'testdata\differential\cases.ndjson'
if ([string]::IsNullOrEmpty($EvidenceJson)) { $EvidenceJson = Join-Path $repositoryRoot 'evidence\differential-summary.json' }
if ([string]::IsNullOrEmpty($EvidenceMarkdown)) { $EvidenceMarkdown = Join-Path $repositoryRoot 'evidence\differential-summary.md' }
$EvidenceJson = [System.IO.Path]::GetFullPath($EvidenceJson)
$EvidenceMarkdown = [System.IO.Path]::GetFullPath($EvidenceMarkdown)
$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$temporaryDirectory = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("micromustache-differential-verify-" + [guid]::NewGuid().ToString('N'))))
if (-not $temporaryDirectory.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Temporary directory escaped the system temp root: $temporaryDirectory"
}
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null

try {
    $firstJson = Join-Path $temporaryDirectory 'first.json'
    $firstMarkdown = Join-Path $temporaryDirectory 'first.md'
    $secondJson = Join-Path $temporaryDirectory 'second.json'
    $secondMarkdown = Join-Path $temporaryDirectory 'second.md'
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $runner -Corpus $corpus -JsonReport $firstJson -MarkdownReport $firstMarkdown -TimeoutSeconds $TimeoutSeconds
    if ($LASTEXITCODE -ne 0) { throw 'First complete differential run failed' }
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $runner -Corpus $corpus -JsonReport $secondJson -MarkdownReport $secondMarkdown -TimeoutSeconds $TimeoutSeconds
    if ($LASTEXITCODE -ne 0) { throw 'Second complete differential run failed' }

    $first = Get-Content -Raw -Encoding UTF8 -LiteralPath $firstJson | ConvertFrom-Json
    $second = Get-Content -Raw -Encoding UTF8 -LiteralPath $secondJson | ConvertFrom-Json
    if ($first.counts.total -lt 150) { throw "Differential corpus has only $($first.counts.total) cases" }
    if ($first.counts.fail -ne 0 -or $second.counts.fail -ne 0) { throw 'A differential run contains FAIL cases' }
    if ($first.deterministicSha256 -cne $second.deterministicSha256) { throw 'Deterministic result hashes differ' }
    if (($first.counts | ConvertTo-Json -Compress) -cne ($second.counts | ConvertTo-Json -Compress)) { throw 'Summary counts differ between runs' }
    if (($first.results | ConvertTo-Json -Depth 100 -Compress) -cne ($second.results | ConvertTo-Json -Depth 100 -Compress)) { throw 'Normalized case results differ between runs' }

    $requiredAPIs = @('render','renderFn','renderFnAsync','compile','compile.render','compile.renderFn','compile.renderFnAsync','compile.sequence','get','getRef','tokenize','renderer.construct','renderer.render','renderer.renderFn','renderer.renderFnAsync','renderer.sequence')
    $actualAPIs = @($first.apis.PSObject.Properties.Name)
    foreach ($api in $requiredAPIs) {
        if ($actualAPIs -notcontains $api) { throw "Corpus does not cover required API operation: $api" }
    }

    Copy-Item -LiteralPath $firstJson -Destination $EvidenceJson -Force
    Copy-Item -LiteralPath $firstMarkdown -Destination $EvidenceMarkdown -Force
    Write-Output "PASS deterministic differential runs=2 cases=$($first.counts.total) pass=$($first.counts.pass) expected_difference=$($first.counts.expectedDifference) skip=$($first.counts.skip) fail=$($first.counts.fail) hash=$($first.deterministicSha256)"
}
finally {
    if (Test-Path -LiteralPath $temporaryDirectory) {
        $resolved = [System.IO.Path]::GetFullPath($temporaryDirectory)
        if (-not $resolved.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove non-temp directory: $resolved"
        }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

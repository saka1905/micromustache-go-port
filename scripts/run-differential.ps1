[CmdletBinding()]
param(
    [string]$Corpus,
    [string]$JsonReport,
    [string]$MarkdownReport,
    [int]$TimeoutSeconds = 30
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
if ([string]::IsNullOrEmpty($Corpus)) { $Corpus = Join-Path $repositoryRoot 'testdata\differential\cases.ndjson' }
if ([string]::IsNullOrEmpty($JsonReport)) { $JsonReport = Join-Path $repositoryRoot 'evidence\differential-summary.json' }
if ([string]::IsNullOrEmpty($MarkdownReport)) { $MarkdownReport = Join-Path $repositoryRoot 'evidence\differential-summary.md' }
$Corpus = [System.IO.Path]::GetFullPath($Corpus)
$JsonReport = [System.IO.Path]::GetFullPath($JsonReport)
$MarkdownReport = [System.IO.Path]::GetFullPath($MarkdownReport)

$node = Get-Command node.exe -ErrorAction Stop
$oracle = Join-Path $repositoryRoot 'oracle\node\oracle.mjs'
$builtUpstream = Join-Path $repositoryRoot 'oracle\upstream\dist\micromustache.cjs'
if (-not (Test-Path -LiteralPath $Corpus -PathType Leaf)) { throw "Corpus not found: $Corpus" }
if (-not (Test-Path -LiteralPath $builtUpstream -PathType Leaf)) { throw 'Node oracle is not prepared' }

$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$temporaryDirectory = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("micromustache-differential-" + [guid]::NewGuid().ToString('N'))))
if (-not $temporaryDirectory.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Temporary directory escaped the system temp root: $temporaryDirectory"
}
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null

try {
    $goRunner = Join-Path $temporaryDirectory 'micromustache-go-oracle.exe'
    $comparator = Join-Path $temporaryDirectory 'micromustache-differential.exe'
    & go build -o $goRunner '.\cmd\micromustache-go-oracle'
    if ($LASTEXITCODE -ne 0) { throw 'Go validation runner build failed' }
    & go build -o $comparator '.\cmd\micromustache-differential'
    if ($LASTEXITCODE -ne 0) { throw 'Differential comparator build failed' }

    $safeDirectory = $repositoryRoot.Replace('\', '/')
    $goCommit = (& git -c "safe.directory=$safeDirectory" rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'git rev-parse HEAD failed' }
    $workingTreeModified = @(git -c "safe.directory=$safeDirectory" status --porcelain).Count -gt 0
    $nodeVersion = (& $node.Path --version).Trim()
    $workingTreeFlag = if ($workingTreeModified) { 'true' } else { 'false' }

    $arguments = @(
        '-repository-root', $repositoryRoot,
        '-corpus', $Corpus,
        '-node', $node.Path,
        '-node-oracle', $oracle,
        '-go-runner', $goRunner,
        '-json-report', $JsonReport,
        '-markdown-report', $MarkdownReport,
        '-upstream-commit', 'da3420db27b7a2fdfbb768811a1280b34952dc95',
        '-go-commit', $goCommit,
        "-working-tree-modified=$workingTreeFlag",
        '-node-version', $nodeVersion,
        '-node-oracle-version', '8.0.3',
        '-command', 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1',
        '-timeout', ("{0}s" -f $TimeoutSeconds)
    )
    & $comparator @arguments
    exit $LASTEXITCODE
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

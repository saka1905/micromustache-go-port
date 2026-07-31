[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Test-UpstreamSnapshot {
    param([string]$Root, [string]$Manifest)

    $snapshotRoot = [System.IO.Path]::GetFullPath($Root).TrimEnd('\', '/')
    $expected = @{}
    $failed = $false
    foreach ($line in @(Get-Content -LiteralPath $Manifest -Encoding UTF8)) {
        $match = [regex]::Match($line, '^(?<Hash>[0-9a-f]{64})  (?<Path>.+)$')
        if (-not $match.Success) {
            Write-Host "FAIL malformed upstream manifest entry: $line"
            $failed = $true
            continue
        }
        $relativePath = $match.Groups['Path'].Value
        if ($expected.ContainsKey($relativePath)) {
            Write-Host "FAIL duplicate upstream manifest entry: $relativePath"
            $failed = $true
            continue
        }
        $expected[$relativePath] = $match.Groups['Hash'].Value
        $candidate = [System.IO.Path]::GetFullPath((Join-Path $snapshotRoot $relativePath.Replace('/', '\')))
        $prefix = $snapshotRoot + [System.IO.Path]::DirectorySeparatorChar
        if (-not $candidate.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
            Write-Host "FAIL upstream path escapes snapshot: $relativePath"
            $failed = $true
            continue
        }
        if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            Write-Host "FAIL missing upstream snapshot file: $relativePath"
            $failed = $true
            continue
        }
        $actual = (Get-FileHash -LiteralPath $candidate -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -cne $expected[$relativePath]) {
            Write-Host "FAIL upstream hash mismatch: $relativePath"
            $failed = $true
            continue
        }
        Write-Host "PASS upstream $relativePath"
    }

    $generatedRoots = @('node_modules', 'dist', 'build', 'coverage', '.npm-cache')
    foreach ($file in @(Get-ChildItem -LiteralPath $snapshotRoot -Recurse -File)) {
        $relativePath = $file.FullName.Substring($snapshotRoot.Length + 1).Replace('\', '/')
        if ($generatedRoots -contains $relativePath.Split('/')[0]) { continue }
        if (-not $expected.ContainsKey($relativePath)) {
            Write-Host "FAIL extra upstream snapshot file: $relativePath"
            $failed = $true
        }
    }
    return -not $failed
}

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$upstreamRoot = Join-Path $repositoryRoot 'oracle\upstream'
$manifestPath = Join-Path $repositoryRoot 'oracle\upstream.sha256'
$originalVerifier = Join-Path $repositoryRoot 'scripts\verify-original-tests.ps1'
$builtEntry = Join-Path $upstreamRoot 'dist\micromustache.cjs'
$smokeRunner = Join-Path $repositoryRoot 'oracle\node\smoke.mjs'
$smokeCases = Join-Path $repositoryRoot 'oracle\cases\smoke.ndjson'
$oraclePath = Join-Path $repositoryRoot 'oracle\node\oracle.mjs'

if (-not (Test-UpstreamSnapshot -Root $upstreamRoot -Manifest $manifestPath)) { exit 1 }
& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $originalVerifier
if ($LASTEXITCODE -ne 0) { Write-Host 'FAIL original-test verification'; exit 1 }

$nodeCommand = Get-Command node.exe -ErrorAction SilentlyContinue
if ($null -eq $nodeCommand) { Write-Host 'FAIL node.exe was not found'; exit 1 }
if (-not (Test-Path -LiteralPath $builtEntry -PathType Leaf)) {
    Write-Host 'FAIL Node oracle is not prepared; run scripts\prepare-node-oracle.ps1 first'
    exit 1
}

& $nodeCommand.Path $smokeRunner --cases $smokeCases --oracle $oraclePath
if ($LASTEXITCODE -ne 0) { Write-Host 'FAIL Node oracle smoke verification'; exit 1 }
if (-not (Test-UpstreamSnapshot -Root $upstreamRoot -Manifest $manifestPath)) { exit 1 }
Write-Host 'PASS Node oracle verification'
exit 0

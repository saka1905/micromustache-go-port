[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$originalRoot = [System.IO.Path]::GetFullPath((Join-Path $repositoryRoot 'tests\original')).TrimEnd('\', '/')
$manifestPath = Join-Path $repositoryRoot 'tests\original.sha256'
$failed = $false
$expected = @{}

if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    Write-Host "FAIL missing manifest: $manifestPath"
    exit 1
}

if (-not (Test-Path -LiteralPath $originalRoot -PathType Container)) {
    Write-Host "FAIL missing original-test directory: $originalRoot"
    exit 1
}

foreach ($line in @(Get-Content -LiteralPath $manifestPath -Encoding UTF8)) {
    $match = [regex]::Match($line, '^(?<Hash>[0-9a-f]{64})  (?<Path>.+)$')
    if (-not $match.Success) {
        Write-Host "FAIL malformed manifest entry: $line"
        $failed = $true
        continue
    }

    $relativePath = $match.Groups['Path'].Value
    $expectedHash = $match.Groups['Hash'].Value
    if ($expected.ContainsKey($relativePath)) {
        Write-Host "FAIL duplicate manifest entry: $relativePath"
        $failed = $true
        continue
    }
    $expected[$relativePath] = $expectedHash

    $platformPath = $relativePath.Replace('/', '\')
    $candidatePath = [System.IO.Path]::GetFullPath((Join-Path $originalRoot $platformPath))
    $requiredPrefix = $originalRoot + [System.IO.Path]::DirectorySeparatorChar
    if (-not $candidatePath.StartsWith($requiredPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        Write-Host "FAIL path outside tests/original: $relativePath"
        $failed = $true
        continue
    }

    if (-not (Test-Path -LiteralPath $candidatePath -PathType Leaf)) {
        Write-Host "FAIL missing: $relativePath"
        $failed = $true
        continue
    }

    $actualHash = (Get-FileHash -LiteralPath $candidatePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -cne $expectedHash) {
        Write-Host "FAIL hash mismatch: $relativePath"
        $failed = $true
        continue
    }

    Write-Host "PASS $relativePath"
}

foreach ($file in @(Get-ChildItem -LiteralPath $originalRoot -Recurse -File)) {
    $relativePath = $file.FullName.Substring($originalRoot.Length + 1).Replace('\', '/')
    if (-not $expected.ContainsKey($relativePath)) {
        Write-Host "FAIL extra file: $relativePath"
        $failed = $true
    }
}

if ($failed) {
    Write-Host 'FAIL original-test verification failed'
    exit 1
}

Write-Host ("PASS all {0} original files match tests/original.sha256" -f $expected.Count)
exit 0

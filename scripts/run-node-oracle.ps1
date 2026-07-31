[CmdletBinding()]
param([string]$InputFile)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$oraclePath = Join-Path $repositoryRoot 'oracle\node\oracle.mjs'
$builtEntry = Join-Path $repositoryRoot 'oracle\upstream\dist\micromustache.cjs'
$nodeCommand = Get-Command node.exe -ErrorAction SilentlyContinue
if ($null -eq $nodeCommand) {
    [Console]::Error.WriteLine('FAIL node.exe was not found')
    exit 1
}
if (-not (Test-Path -LiteralPath $builtEntry -PathType Leaf)) {
    [Console]::Error.WriteLine('FAIL Node oracle is not prepared; run scripts\prepare-node-oracle.ps1 first')
    exit 1
}

if ([string]::IsNullOrEmpty($InputFile)) {
    & $nodeCommand.Path $oraclePath
} else {
    $resolvedInput = (Resolve-Path -LiteralPath $InputFile -ErrorAction Stop).Path
    & $nodeCommand.Path $oraclePath --input-file $resolvedInput
}
exit $LASTEXITCODE

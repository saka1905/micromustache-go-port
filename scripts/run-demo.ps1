[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-CapturedProcess {
    param(
        [string]$Name,
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$WorkingDirectory,
        [int]$TimeoutSeconds = 60
    )

    $utf8 = New-Object System.Text.UTF8Encoding($false, $true)
    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $FilePath
    $startInfo.Arguments = (($Arguments | ForEach-Object { '"' + $_.Replace('"', '\"') + '"' }) -join ' ')
    $startInfo.WorkingDirectory = $WorkingDirectory
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.StandardOutputEncoding = $utf8
    $startInfo.StandardErrorEncoding = $utf8

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo
    if (-not $process.Start()) { throw "$Name could not start" }
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
        $process.Kill()
        $process.WaitForExit()
        throw "$Name timed out after $TimeoutSeconds seconds"
    }
    $process.WaitForExit()
    return [pscustomobject]@{
        ExitCode = $process.ExitCode
        Stdout = $stdoutTask.Result
        Stderr = $stderrTask.Result
    }
}

function Assert-Success {
    param([string]$Name, $Result, [switch]$AllowStdout)
    if ($Result.ExitCode -ne 0) { throw "$Name exited $($Result.ExitCode): $($Result.Stderr)" }
    if (-not [string]::IsNullOrEmpty($Result.Stderr)) { throw "$Name wrote stderr: $($Result.Stderr)" }
    if (-not $AllowStdout -and -not [string]::IsNullOrEmpty($Result.Stdout)) { throw "$Name wrote unexpected stdout: $($Result.Stdout)" }
}

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$goCommand = Get-Command go.exe -ErrorAction Stop
$powerShellCommand = Get-Command powershell.exe -ErrorAction Stop
$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$temporaryDirectory = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("micromustache-demo-" + [guid]::NewGuid().ToString('N'))))
$temporaryPrefix = $temporaryRoot + [System.IO.Path]::DirectorySeparatorChar
if (-not $temporaryDirectory.StartsWith($temporaryPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Temporary directory escaped the system temp root: $temporaryDirectory"
}
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null

try {
    $version = Invoke-CapturedProcess -Name 'Go version' -FilePath $goCommand.Path -Arguments @('version') -WorkingDirectory $repositoryRoot
    Assert-Success -Name 'Go version' -Result $version -AllowStdout
    if ([string]::IsNullOrWhiteSpace($version.Stdout)) { throw 'Go version produced no output' }

    $demoBinary = Join-Path $temporaryDirectory 'micromustache-demo.exe'
    $build = Invoke-CapturedProcess -Name 'Demo build' -FilePath $goCommand.Path -Arguments @('build', '-o', $demoBinary, '.\cmd\micromustache-demo') -WorkingDirectory $repositoryRoot
    Assert-Success -Name 'Demo build' -Result $build
    if (-not (Test-Path -LiteralPath $demoBinary -PathType Leaf)) { throw 'Demo build produced no binary' }

    $first = Invoke-CapturedProcess -Name 'Demo run 1' -FilePath $demoBinary -Arguments @() -WorkingDirectory $temporaryDirectory
    Assert-Success -Name 'Demo run 1' -Result $first -AllowStdout
    $second = Invoke-CapturedProcess -Name 'Demo run 2' -FilePath $demoBinary -Arguments @() -WorkingDirectory $temporaryDirectory
    Assert-Success -Name 'Demo run 2' -Result $second -AllowStdout

    $utf8 = New-Object System.Text.UTF8Encoding($false, $true)
    $firstBytes = $utf8.GetBytes($first.Stdout)
    $secondBytes = $utf8.GetBytes($second.Stdout)
    [System.IO.File]::WriteAllBytes((Join-Path $temporaryDirectory 'run-1.txt'), $firstBytes)
    [System.IO.File]::WriteAllBytes((Join-Path $temporaryDirectory 'run-2.txt'), $secondBytes)
    if ([System.Convert]::ToBase64String($firstBytes) -cne [System.Convert]::ToBase64String($secondBytes)) {
        throw 'Demo output differs byte-for-byte between runs'
    }

    $requiredSections = @(
        '[1/6] Basic Render',
        '[2/6] Tokenize',
        '[3/6] Get and GetRef',
        '[4/6] Compile and Renderer Reuse',
        '[5/6] Synchronous Resolver',
        '[6/6] Asynchronous Resolver'
    )
    $lastIndex = -1
    foreach ($heading in $requiredSections) {
        $matches = [regex]::Matches($first.Stdout, [regex]::Escape($heading))
        if ($matches.Count -ne 1) { throw "Required section count is not one: $heading" }
        if ($matches[0].Index -le $lastIndex) { throw "Required section is out of order: $heading" }
        $lastIndex = $matches[0].Index
    }
    if ([regex]::Matches($first.Stdout, '(?m)^DEMO_STATUS: PASS\r?$').Count -ne 1) {
        throw 'Demo final PASS line is missing or duplicated'
    }

    $original = Invoke-CapturedProcess -Name 'Original-test integrity' -FilePath $powerShellCommand.Path -Arguments @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', (Join-Path $repositoryRoot 'scripts\verify-original-tests.ps1')) -WorkingDirectory $repositoryRoot
    Assert-Success -Name 'Original-test integrity' -Result $original -AllowStdout
    $originalMatch = [regex]::Match($original.Stdout, 'PASS all (?<Count>\d+) original files match tests/original\.sha256')
    if (-not $originalMatch.Success) { throw 'Original-test verifier did not report its final count' }

    $differential = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'evidence\differential-summary.json') | ConvertFrom-Json
    $benchmark = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'evidence\benchmark-summary.json') | ConvertFrom-Json
    if ($differential.schemaVersion -ne 1 -or $differential.counts.total -ne 218 -or $differential.counts.fail -ne 0) {
        throw 'Differential evidence has unexpected schema, total, or FAIL count'
    }
    $rawSamples = (@($benchmark.workloads | ForEach-Object { @($_.rawSamples).Count }) | Measure-Object -Sum).Sum
    $correctWorkloads = @($benchmark.workloads | Where-Object { $_.correctness.status -ceq 'PASS' }).Count
    if ($benchmark.schemaVersion -ne 1 -or $benchmark.workloads.Count -ne 26 -or $rawSamples -ne 728 -or $correctWorkloads -ne 26 -or $benchmark.correctnessStatus -cne 'PASS') {
        throw 'Benchmark evidence has unexpected schema, sample count, or correctness status'
    }
    $workloadHash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $repositoryRoot 'testdata\benchmark\workloads.json')).Hash.ToLowerInvariant()
    if ($workloadHash -cne $benchmark.workloadSha256) { throw 'Benchmark workload hash does not match evidence' }

    Write-Output ("Go runtime: {0}" -f $version.Stdout.Trim())
    Write-Output 'Demo output:'
    [Console]::Out.Write($first.Stdout)
    Write-Output 'Determinism: PASS (2 runs, byte-for-byte identical)'
    Write-Output ''
    Write-Output 'Evidence summary'
    Write-Output ("original upstream tests preserved: PASS ({0}/{0} SHA-256)" -f $originalMatch.Groups['Count'].Value)
    Write-Output ("differential: total={0} pass={1} approved={2} skip={3} fail={4}" -f $differential.counts.total, $differential.counts.pass, $differential.counts.expectedDifference, $differential.counts.skip, $differential.counts.fail)
    Write-Output ("benchmark: workloads={0} raw_samples={1} correctness=PASS ({2}/{0})" -f $benchmark.workloads.Count, $rawSamples, $correctWorkloads)
    Write-Output ''
    Write-Output 'Full differential validation:'
    Write-Output 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1'
    Write-Output 'Full benchmark reproduction:'
    Write-Output 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-benchmark.ps1'
    Write-Output ''
    Write-Output 'WALKTHROUGH_STATUS: PASS'
}
finally {
    if (Test-Path -LiteralPath $temporaryDirectory) {
        $resolved = [System.IO.Path]::GetFullPath($temporaryDirectory)
        if (-not $resolved.StartsWith($temporaryPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove non-temp directory: $resolved"
        }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

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
        [int]$TimeoutSeconds = 180
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

function Assert-ProcessSuccess {
    param([string]$Name, $Result, [switch]$AllowStdout)
    if ($Result.ExitCode -ne 0) { throw "$Name exited $($Result.ExitCode): $($Result.Stderr)" }
    if (-not [string]::IsNullOrEmpty($Result.Stderr)) { throw "$Name wrote stderr: $($Result.Stderr)" }
    if (-not $AllowStdout -and -not [string]::IsNullOrEmpty($Result.Stdout)) {
        throw "$Name wrote unexpected stdout: $($Result.Stdout)"
    }
}

function Test-Sha256Manifest {
    param(
        [string]$Root,
        [string]$Manifest,
        [string[]]$ExcludedTopLevel = @()
    )

    $manifestRoot = [System.IO.Path]::GetFullPath($Root).TrimEnd('\', '/')
    $requiredPrefix = $manifestRoot + [System.IO.Path]::DirectorySeparatorChar
    $expected = @{}
    foreach ($line in @(Get-Content -Encoding UTF8 -LiteralPath $Manifest)) {
        $match = [regex]::Match($line, '^(?<Hash>[0-9a-f]{64})  (?<Path>.+)$')
        if (-not $match.Success) { throw "Malformed manifest entry in ${Manifest}: $line" }
        $relativePath = $match.Groups['Path'].Value
        if ($expected.ContainsKey($relativePath)) { throw "Duplicate manifest entry: $relativePath" }
        $candidate = [System.IO.Path]::GetFullPath((Join-Path $manifestRoot $relativePath.Replace('/', '\')))
        if (-not $candidate.StartsWith($requiredPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
            throw "Manifest path escapes root: $relativePath"
        }
        if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) { throw "Missing manifest file: $relativePath" }
        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $candidate).Hash.ToLowerInvariant()
        if ($actual -cne $match.Groups['Hash'].Value) { throw "Manifest hash mismatch: $relativePath" }
        $expected[$relativePath] = $actual
    }

    foreach ($file in @(Get-ChildItem -LiteralPath $manifestRoot -Recurse -File)) {
        $relativePath = $file.FullName.Substring($manifestRoot.Length + 1).Replace('\', '/')
        if ($ExcludedTopLevel -contains $relativePath.Split('/')[0]) { continue }
        if (-not $expected.ContainsKey($relativePath)) { throw "File is missing from manifest: $relativePath" }
    }
    return $expected.Count
}

function Test-MarkdownLinks {
    param([string]$RepositoryRoot, [string[]]$Files)

    $count = 0
    foreach ($relativeFile in $Files) {
        $file = Join-Path $RepositoryRoot $relativeFile.Replace('/', '\')
        $text = Get-Content -Raw -Encoding UTF8 -LiteralPath $file
        foreach ($match in [regex]::Matches($text, '\[[^\]]+\]\((?<Target>[^)]+)\)')) {
            $target = $match.Groups['Target'].Value.Trim('<', '>')
            if ($target -match '^(https?://|mailto:|#)') { continue }
            $target = ($target -split '#', 2)[0]
            if ([string]::IsNullOrEmpty($target)) { continue }
            $candidate = [System.IO.Path]::GetFullPath((Join-Path (Split-Path -Parent $file) $target))
            if (-not (Test-Path -LiteralPath $candidate)) { throw "Broken local Markdown link in ${relativeFile}: $target" }
            $count++
        }
    }
    return $count
}

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$requiredFiles = @(
    '.gitattributes',
    'README.md',
    'DECISIONS.md',
    'LICENSE',
    'THIRD_PARTY_NOTICES.md',
    'go.mod',
    'docs\API_MAPPING.md',
    'docs\DIFFERENTIAL_TESTING.md',
    'docs\BENCHMARKING.md',
    'docs\DEMO.md',
    'docs\SUBMISSION_DRAFT.md',
    'docs\SUBMISSION_COMPLIANCE.md',
    'evidence\differential-summary.json',
    'evidence\benchmark-summary.json',
    'evidence\demo-output.txt',
    'tests\original.sha256',
    'oracle\upstream.sha256',
    'scripts\prepare-node-oracle.ps1',
    'scripts\verify-differential.ps1',
    'scripts\verify-benchmark.ps1',
    'scripts\run-demo.ps1'
)
foreach ($relativePath in $requiredFiles) {
    if (-not (Test-Path -LiteralPath (Join-Path $repositoryRoot $relativePath))) { throw "Required file is missing: $relativePath" }
}

$moduleLines = @(Get-Content -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'go.mod'))
if ($moduleLines.Count -lt 1 -or $moduleLines[0] -cne 'module github.com/saka1905/micromustache-go-port') {
    throw 'go.mod module path is not the documented module'
}
if (Test-Path -LiteralPath (Join-Path $repositoryRoot 'go.sum')) { throw 'go.sum must not exist for this dependency-free module' }

$goCommand = Get-Command go.exe -ErrorAction Stop
$gitCommand = Get-Command git.exe -ErrorAction Stop
$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$temporaryDirectory = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("micromustache-submission-" + [guid]::NewGuid().ToString('N'))))
$temporaryPrefix = $temporaryRoot + [System.IO.Path]::DirectorySeparatorChar
if (-not $temporaryDirectory.StartsWith($temporaryPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Temporary directory escaped the system temp root: $temporaryDirectory"
}
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
$completed = $false

try {
    $version = Invoke-CapturedProcess -Name 'Go version' -FilePath $goCommand.Path -Arguments @('version') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Go version' -Result $version -AllowStdout

    $build = Invoke-CapturedProcess -Name 'Product build' -FilePath $goCommand.Path -Arguments @('build', './...') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Product build' -Result $build
    $tests = Invoke-CapturedProcess -Name 'Go tests' -FilePath $goCommand.Path -Arguments @('test', '-count=1', './...') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Go tests' -Result $tests -AllowStdout
    $vet = Invoke-CapturedProcess -Name 'Go vet' -FilePath $goCommand.Path -Arguments @('vet', './...') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Go vet' -Result $vet
    $packages = Invoke-CapturedProcess -Name 'Go package list' -FilePath $goCommand.Path -Arguments @('list', './...') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Go package list' -Result $packages -AllowStdout
    if ($packages.Stdout -notmatch '(?m)^github\.com/saka1905/micromustache-go-port/cmd/micromustache-demo\r?$') {
        throw 'Go package list does not contain the demo command'
    }
    $modules = Invoke-CapturedProcess -Name 'Go module list' -FilePath $goCommand.Path -Arguments @('list', '-m', 'all') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Go module list' -Result $modules -AllowStdout
    $moduleList = @($modules.Stdout.Trim() -split '\r?\n')
    if ($moduleList.Count -ne 1 -or $moduleList[0] -cne 'github.com/saka1905/micromustache-go-port') {
        throw 'External Go module dependency detected'
    }

    $demoBinary = Join-Path $temporaryDirectory 'micromustache-demo.exe'
    $demoBuild = Invoke-CapturedProcess -Name 'Demo build' -FilePath $goCommand.Path -Arguments @('build', '-o', $demoBinary, '.\cmd\micromustache-demo') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Demo build' -Result $demoBuild
    $demo = Invoke-CapturedProcess -Name 'Demo run' -FilePath $demoBinary -Arguments @() -WorkingDirectory $temporaryDirectory
    Assert-ProcessSuccess -Name 'Demo run' -Result $demo -AllowStdout
    if ([regex]::Matches($demo.Stdout, '(?m)^DEMO_STATUS: PASS\r?$').Count -ne 1) {
        throw 'Demo final PASS line is missing or duplicated'
    }
    $utf8 = New-Object System.Text.UTF8Encoding($false, $true)
    $actualDemo = $utf8.GetBytes($demo.Stdout)
    $expectedDemo = [System.IO.File]::ReadAllBytes((Join-Path $repositoryRoot 'evidence\demo-output.txt'))
    if ([System.Convert]::ToBase64String($actualDemo) -cne [System.Convert]::ToBase64String($expectedDemo)) {
        throw 'Demo output does not match tracked evidence byte-for-byte'
    }

    $originalCount = Test-Sha256Manifest -Root (Join-Path $repositoryRoot 'tests\original') -Manifest (Join-Path $repositoryRoot 'tests\original.sha256')
    if ($originalCount -ne 16) { throw "Expected 16 original files, got $originalCount" }
    $upstreamCount = Test-Sha256Manifest -Root (Join-Path $repositoryRoot 'oracle\upstream') -Manifest (Join-Path $repositoryRoot 'oracle\upstream.sha256') -ExcludedTopLevel @('node_modules', 'dist', 'build', 'coverage', '.npm-cache')
    if ($upstreamCount -ne 12) { throw "Expected 12 upstream snapshot files, got $upstreamCount" }

    $differential = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'evidence\differential-summary.json') | ConvertFrom-Json
    if ($differential.schemaVersion -ne 1 -or $differential.counts.total -ne 218 -or $differential.counts.pass -ne 202 -or $differential.counts.expectedDifference -ne 13 -or $differential.counts.skip -ne 3 -or $differential.counts.fail -ne 0 -or $differential.results.Count -ne 218) {
        throw 'Differential evidence counts or schema are invalid'
    }
    if ($differential.deterministicSha256 -cne '46b9c5498cde0faf7203ee6426679cbbc964e667c2fe36ba70837abf9eddef4b') {
        throw 'Differential deterministic result hash is invalid'
    }

    $benchmark = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'evidence\benchmark-summary.json') | ConvertFrom-Json
    $rawSamples = (@($benchmark.workloads | ForEach-Object { @($_.rawSamples).Count }) | Measure-Object -Sum).Sum
    $correctWorkloads = @($benchmark.workloads | Where-Object { $_.correctness.status -ceq 'PASS' }).Count
    if ($benchmark.schemaVersion -ne 1 -or $benchmark.workloads.Count -ne 26 -or $rawSamples -ne 728 -or $correctWorkloads -ne 26 -or $benchmark.correctnessStatus -cne 'PASS') {
        throw 'Benchmark evidence counts or correctness are invalid'
    }
    foreach ($workload in $benchmark.workloads) {
        foreach ($runtime in @('node', 'go')) {
            foreach ($round in @(1, 2)) {
                if (@($workload.rawSamples | Where-Object { $_.runtime -ceq $runtime -and $_.round -eq $round }).Count -ne 7) {
                    throw "Benchmark runtime round is incomplete: $($workload.id) $runtime round $round"
                }
            }
        }
    }
    $workloadHash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $repositoryRoot 'testdata\benchmark\workloads.json')).Hash.ToLowerInvariant()
    if ($workloadHash -cne $benchmark.workloadSha256) { throw 'Benchmark workload hash does not match evidence' }

    $readme = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'README.md')
    foreach ($requiredLink in @('docs/SUBMISSION_DRAFT.md', 'docs/SUBMISSION_COMPLIANCE.md', 'docs/API_MAPPING.md', 'docs/DIFFERENTIAL_TESTING.md', 'docs/BENCHMARKING.md', 'docs/DEMO.md', 'THIRD_PARTY_NOTICES.md')) {
        if (-not $readme.Contains($requiredLink)) { throw "README does not link required documentation: $requiredLink" }
    }
    foreach ($requiredCommand in @('go build ./...', 'go run ./cmd/micromustache-demo', 'scripts/verify-submission.ps1', 'scripts/prepare-node-oracle.ps1', 'scripts/verify-differential.ps1', 'scripts/verify-benchmark.ps1')) {
        if (-not $readme.Contains($requiredCommand)) { throw "README does not contain required command: $requiredCommand" }
    }
    $markdownResult = Invoke-CapturedProcess -Name 'Tracked Markdown list' -FilePath $gitCommand.Path -Arguments @('ls-files', '*.md') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Tracked Markdown list' -Result $markdownResult -AllowStdout
    $markdownFiles = @(@($markdownResult.Stdout.Trim() -split '\r?\n' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }) + @('docs/SUBMISSION_DRAFT.md', 'docs/SUBMISSION_COMPLIANCE.md') | Sort-Object -Unique)
    $linkCount = Test-MarkdownLinks -RepositoryRoot $repositoryRoot -Files $markdownFiles

    $trackedResult = Invoke-CapturedProcess -Name 'Tracked file list' -FilePath $gitCommand.Path -Arguments @('ls-files') -WorkingDirectory $repositoryRoot
    Assert-ProcessSuccess -Name 'Tracked file list' -Result $trackedResult -AllowStdout
    $trackedGenerated = @($trackedResult.Stdout -split '\r?\n' | Where-Object { $_ -match '(^|/)(node_modules|dist)/' })
    if ($trackedGenerated.Count -ne 0) { throw "Tracked generated Node artifact found: $($trackedGenerated -join ', ')" }

    $runtimeFiles = @(Get-ChildItem -LiteralPath $repositoryRoot -File -Filter '*.go') + @((Get-Item -LiteralPath (Join-Path $repositoryRoot 'cmd\micromustache-demo\main.go')))
    foreach ($file in $runtimeFiles) {
        $source = Get-Content -Raw -Encoding UTF8 -LiteralPath $file.FullName
        if ($source -match '"os/exec"|exec\.Command|os\.StartProcess|syscall\.StartProcess') {
            throw "Runtime subprocess launch path found: $($file.FullName)"
        }
    }

    $projectLicense = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'LICENSE')
    $upstreamLicense = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'third_party\micromustache\LICENSE')
    $notices = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repositoryRoot 'THIRD_PARTY_NOTICES.md')
    if ($projectLicense -notmatch '^MIT License' -or $upstreamLicense -notmatch '^MIT License') { throw 'MIT license text is missing' }
    foreach ($noticeFact in @('alexewerlof/micromustache', 'da3420db27b7a2fdfbb768811a1280b34952dc95', '8.0.3', 'Copyright (c) 2019', 'Alex Ewerl', 'MIT License')) {
        if (-not $notices.Contains($noticeFact)) { throw "Third-party notice is missing: $noticeFact" }
    }

    Write-Output ("PASS Go runtime: {0}" -f $version.Stdout.Trim())
    Write-Output 'PASS one-command product build: go build ./...'
    Write-Output 'PASS Go tests, vet, package list, and dependency boundary'
    Write-Output 'PASS Go-only demo and tracked output evidence'
    Write-Output ("PASS protected manifests: original={0}/16 upstream={1}/12" -f $originalCount, $upstreamCount)
    Write-Output ("PASS differential evidence: total={0} fail={1} hash={2}" -f $differential.counts.total, $differential.counts.fail, $differential.deterministicSha256)
    Write-Output ("PASS benchmark evidence: workloads={0} rawSamples={1} correctness={2}/26" -f $benchmark.workloads.Count, $rawSamples, $correctWorkloads)
    Write-Output ("PASS documentation and license links: localLinks={0}" -f $linkCount)
    Write-Output 'PASS tracked generated artifacts=0 and runtime subprocess paths=0'
    $completed = $true
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

if (-not $completed) { throw 'Submission verification did not complete' }
Write-Output 'SUBMISSION_VERIFICATION: PASS'

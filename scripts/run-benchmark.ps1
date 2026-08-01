[CmdletBinding()]
param(
    [string]$Workloads,
    [string]$JsonReport,
    [string]$MarkdownReport,
    [int]$Warmup = 3,
    [int]$Samples = 7,
    [int]$MinDurationMs = 200,
    [long]$MaxIterations = 16777216,
    [int]$ProcessTimeoutSeconds = 300
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-CapturedProcess {
    param(
        [string]$Name,
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$StdoutPath,
        [string]$StderrPath,
        [int]$TimeoutSeconds
    )
    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $FilePath
    $startInfo.Arguments = (($Arguments | ForEach-Object { '"' + $_.Replace('"', '\"') + '"' }) -join ' ')
    $startInfo.WorkingDirectory = $repositoryRoot
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
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
    $stdout = $stdoutTask.Result
    $stderr = $stderrTask.Result
    $utf8 = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($StdoutPath, $stdout, $utf8)
    [System.IO.File]::WriteAllText($StderrPath, $stderr, $utf8)
    if ($process.ExitCode -ne 0) { throw "$Name exited $($process.ExitCode): $stderr" }
    if (-not [string]::IsNullOrWhiteSpace($stderr)) { throw "$Name wrote stderr: $stderr" }
    if (-not (Test-Path -LiteralPath $StdoutPath -PathType Leaf) -or (Get-Item -LiteralPath $StdoutPath).Length -eq 0) { throw "$Name produced no stdout" }
}

$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
if ([string]::IsNullOrEmpty($Workloads)) { $Workloads = Join-Path $repositoryRoot 'testdata\benchmark\workloads.json' }
if ([string]::IsNullOrEmpty($JsonReport)) { $JsonReport = Join-Path $repositoryRoot 'evidence\benchmark-summary.json' }
if ([string]::IsNullOrEmpty($MarkdownReport)) { $MarkdownReport = Join-Path $repositoryRoot 'evidence\benchmark-summary.md' }
$Workloads = [System.IO.Path]::GetFullPath($Workloads)
$JsonReport = [System.IO.Path]::GetFullPath($JsonReport)
$MarkdownReport = [System.IO.Path]::GetFullPath($MarkdownReport)
if (-not (Test-Path -LiteralPath $Workloads -PathType Leaf)) { throw "Workloads not found: $Workloads" }
if (-not (Test-Path -LiteralPath (Join-Path $repositoryRoot 'oracle\upstream\dist\micromustache.cjs') -PathType Leaf)) { throw 'Node oracle is not prepared' }
if ($Warmup -lt 3 -or $Samples -lt 7 -or $MinDurationMs -le 0 -or $MaxIterations -le 0 -or $ProcessTimeoutSeconds -le 0) { throw 'Invalid benchmark configuration' }

$node = Get-Command node.exe -ErrorAction Stop
$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$temporaryDirectory = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("micromustache-benchmark-" + [guid]::NewGuid().ToString('N'))))
if (-not $temporaryDirectory.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) { throw "Temporary directory escaped system temp: $temporaryDirectory" }
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null

try {
    $goRunner = Join-Path $temporaryDirectory 'micromustache-go-benchmark.exe'
    $reporter = Join-Path $temporaryDirectory 'micromustache-benchmark-report.exe'
    & go build -o $goRunner '.\cmd\micromustache-go-benchmark'
    if ($LASTEXITCODE -ne 0) { throw 'Go benchmark runner build failed' }
    & go build -o $reporter '.\cmd\micromustache-benchmark-report'
    if ($LASTEXITCODE -ne 0) { throw 'Benchmark reporter build failed' }

    $common = @('--workloads', $Workloads, '--warmup', "$Warmup", '--samples', "$Samples", '--min-duration-ms', "$MinDurationMs", '--max-iterations', "$MaxIterations", '--process-timeout-seconds', "$ProcessTimeoutSeconds")
    $goCommon = @($common | ForEach-Object { $_ -replace '^--', '-' })
    $nodeScript = Join-Path $repositoryRoot 'oracle\node\benchmark.mjs'
    $validationNode = Join-Path $temporaryDirectory 'validation-node.json'
    $validationGo = Join-Path $temporaryDirectory 'validation-go.json'
    $nodeValidateArguments = @($nodeScript, '--mode', 'validate') + $common
    $goValidateArguments = @('-mode', 'validate') + $goCommon
    Invoke-CapturedProcess -Name 'Node correctness runner' -FilePath $node.Path -Arguments $nodeValidateArguments -StdoutPath $validationNode -StderrPath (Join-Path $temporaryDirectory 'validation-node.stderr') -TimeoutSeconds $ProcessTimeoutSeconds
    Invoke-CapturedProcess -Name 'Go correctness runner' -FilePath $goRunner -Arguments $goValidateArguments -StdoutPath $validationGo -StderrPath (Join-Path $temporaryDirectory 'validation-go.stderr') -TimeoutSeconds $ProcessTimeoutSeconds
    Invoke-CapturedProcess -Name 'Correctness comparator' -FilePath $reporter -Arguments @('-mode','validate','-workloads',$Workloads,'-validation-node',$validationNode,'-validation-go',$validationGo) -StdoutPath (Join-Path $temporaryDirectory 'validation-report.stdout') -StderrPath (Join-Path $temporaryDirectory 'validation-report.stderr') -TimeoutSeconds 30

    $round1Node = Join-Path $temporaryDirectory 'round1-node.json'
    $round1Go = Join-Path $temporaryDirectory 'round1-go.json'
    $round2Node = Join-Path $temporaryDirectory 'round2-node.json'
    $round2Go = Join-Path $temporaryDirectory 'round2-go.json'
    $nodeBenchmarkArguments = @($nodeScript, '--mode', 'benchmark') + $common
    $goBenchmarkArguments = @('-mode', 'benchmark') + $goCommon
    Invoke-CapturedProcess -Name 'Round 1 Node benchmark' -FilePath $node.Path -Arguments $nodeBenchmarkArguments -StdoutPath $round1Node -StderrPath (Join-Path $temporaryDirectory 'round1-node.stderr') -TimeoutSeconds $ProcessTimeoutSeconds
    Invoke-CapturedProcess -Name 'Round 1 Go benchmark' -FilePath $goRunner -Arguments $goBenchmarkArguments -StdoutPath $round1Go -StderrPath (Join-Path $temporaryDirectory 'round1-go.stderr') -TimeoutSeconds $ProcessTimeoutSeconds
    Invoke-CapturedProcess -Name 'Round 2 Go benchmark' -FilePath $goRunner -Arguments $goBenchmarkArguments -StdoutPath $round2Go -StderrPath (Join-Path $temporaryDirectory 'round2-go.stderr') -TimeoutSeconds $ProcessTimeoutSeconds
    Invoke-CapturedProcess -Name 'Round 2 Node benchmark' -FilePath $node.Path -Arguments $nodeBenchmarkArguments -StdoutPath $round2Node -StderrPath (Join-Path $temporaryDirectory 'round2-node.stderr') -TimeoutSeconds $ProcessTimeoutSeconds

    $cpuModel = 'unavailable'
    $installedMemory = 'unavailable'
    try { $cpuModel = (@(Get-CimInstance Win32_Processor -ErrorAction Stop)[0].Name).Trim() } catch {}
    try { $installedMemory = "$([long](Get-CimInstance Win32_ComputerSystem -ErrorAction Stop).TotalPhysicalMemory) bytes" } catch {}
    $powerPlan = 'unavailable'
    try { $powerPlanOutput = @(& powercfg.exe /getactivescheme 2>$null); if ($LASTEXITCODE -eq 0 -and $powerPlanOutput.Count -gt 0) { $powerPlan = ($powerPlanOutput -join ' ').Trim() } } catch {}
    $goVersion = (& go env GOVERSION).Trim()
    $nodeVersion = (& $node.Path --version).Trim()
    $architecture = if ($env:PROCESSOR_ARCHITECTURE -eq 'AMD64') { 'amd64' } else { $env:PROCESSOR_ARCHITECTURE.ToLowerInvariant() }
    $environmentObject = [ordered]@{goVersion=$goVersion;nodeVersion=$nodeVersion;os='windows';architecture=$architecture;cpuModel=$cpuModel;logicalProcessors=[int]$env:NUMBER_OF_PROCESSORS;installedMemory=$installedMemory;powerPlan=$powerPlan}
    $environmentPath = Join-Path $temporaryDirectory 'environment.json'
    [System.IO.File]::WriteAllText($environmentPath, ($environmentObject | ConvertTo-Json -Compress), (New-Object System.Text.UTF8Encoding($false)))

    $safeDirectory = $repositoryRoot.Replace('\','/')
    $repositoryCommit = (& git -c "safe.directory=$safeDirectory" rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'git rev-parse failed' }
    $dirty = @(git -c "safe.directory=$safeDirectory" status --porcelain).Count -gt 0
    $dirtyFlag = if ($dirty) { 'true' } else { 'false' }
    $reportArguments = @('-mode','report','-workloads',$Workloads,'-validation-node',$validationNode,'-validation-go',$validationGo,'-round1-node',$round1Node,'-round1-go',$round1Go,'-round2-node',$round2Node,'-round2-go',$round2Go,'-environment',$environmentPath,'-repository-commit',$repositoryCommit,"-repository-dirty=$dirtyFlag",'-json-report',$JsonReport,'-markdown-report',$MarkdownReport)
    Invoke-CapturedProcess -Name 'Benchmark report generator' -FilePath $reporter -Arguments $reportArguments -StdoutPath (Join-Path $temporaryDirectory 'report.stdout') -StderrPath (Join-Path $temporaryDirectory 'report.stderr') -TimeoutSeconds 30
    Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $temporaryDirectory 'report.stdout')
}
finally {
    if (Test-Path -LiteralPath $temporaryDirectory) {
        $resolved = [System.IO.Path]::GetFullPath($temporaryDirectory)
        if (-not $resolved.StartsWith($temporaryRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) { throw "Refusing to remove non-temp directory: $resolved" }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

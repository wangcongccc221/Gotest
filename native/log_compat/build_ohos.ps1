$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = Resolve-Path (Join-Path $scriptDir '..\..')

if (-not $env:OHOS_SDK) {
  $env:OHOS_SDK = 'D:\harmSdk\command-line-tools\sdk\default\openharmony'
}

$ohosArch = if ($env:OHOS_ARCH) { $env:OHOS_ARCH } else { 'arm64-v8a' }
$cmake = Join-Path $env:OHOS_SDK 'native\build-tools\cmake\bin\cmake.exe'
$ninja = Join-Path $env:OHOS_SDK 'native\build-tools\cmake\bin\ninja.exe'
$toolchain = Join-Path $env:OHOS_SDK 'native\build\cmake\ohos.toolchain.cmake'
$buildDir = Join-Path $scriptDir "build\ninja-$ohosArch"
$outDir = Join-Path $rootDir "entry\libs\$ohosArch"

& $cmake -G Ninja `
  -S $scriptDir `
  -B $buildDir `
  "-DCMAKE_MAKE_PROGRAM=$ninja" `
  "-DCMAKE_TOOLCHAIN_FILE=$toolchain" `
  "-DOHOS_ARCH=$ohosArch" `
  -DCMAKE_BUILD_TYPE=Release

if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& $cmake --build $buildDir --config Release
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$logSo = Get-ChildItem $buildDir -Recurse -Filter liblog.so | Select-Object -First 1
if (-not $logSo) {
  throw "liblog.so was not produced under $buildDir"
}

New-Item -ItemType Directory -Force $outDir | Out-Null
Copy-Item -Force $logSo.FullName (Join-Path $outDir 'liblog.so')

Write-Output "Built $(Join-Path $outDir 'liblog.so')"

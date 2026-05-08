$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = Resolve-Path (Join-Path $scriptDir '..\..')

if (-not $env:OHOS_SDK) {
  $env:OHOS_SDK = 'D:\harmSdk\command-line-tools\sdk\default\openharmony'
}

$ohosArch = if ($env:OHOS_ARCH) { $env:OHOS_ARCH } else { 'arm64-v8a' }

switch ($ohosArch) {
  'arm64-v8a' {
    $env:GOARCH = 'arm64'
    $target = 'aarch64-linux-ohos'
    $sysrootLibDir = 'aarch64-linux-ohos'
  }
  default {
    throw "Unsupported OHOS_ARCH=$ohosArch. This POC currently supports arm64-v8a."
  }
}

$logLib = Join-Path $rootDir "entry\libs\$ohosArch\liblog.so"
if (-not (Test-Path $logLib)) {
  & (Join-Path $rootDir 'native\log_compat\build_ohos.ps1')
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$env:GOOS = 'android'
$env:CGO_ENABLED = '1'
$env:CGO_CFLAGS_ALLOW = '.*'
$env:CGO_LDFLAGS_ALLOW = '.*'

$llvmConfig = Join-Path $env:OHOS_SDK 'native\llvm\bin\llvm-config.exe'
$llvmCflags = (& $llvmConfig --cflags) -join ' '
$env:CGO_CFLAGS = "$llvmCflags --target=$target --sysroot=$env:OHOS_SDK/native/sysroot -I$rootDir/native/log_compat/include"
$env:CGO_LDFLAGS = "--target=$target --sysroot=$env:OHOS_SDK/native/sysroot -fuse-ld=lld -L$rootDir/entry/libs/$ohosArch -L$env:OHOS_SDK/native/sysroot/usr/lib/$sysrootLibDir"

$env:CC = Join-Path $env:OHOS_SDK 'native\llvm\bin\clang.exe'
$env:CXX = Join-Path $env:OHOS_SDK 'native\llvm\bin\clang++.exe'
$env:AR = Join-Path $env:OHOS_SDK 'native\llvm\bin\llvm-ar.exe'
$env:LD = Join-Path $env:OHOS_SDK 'native\llvm\bin\ld.lld.exe'

$outDir = Join-Path $rootDir "entry\libs\$ohosArch"
New-Item -ItemType Directory -Force $outDir | Out-Null

Push-Location $scriptDir
go build -tags netgo -buildmode=c-shared -trimpath -ldflags='-s -w' -o (Join-Path $outDir 'libohos.so') .
$goCode = $LASTEXITCODE
Pop-Location
if ($goCode -ne 0) { exit $goCode }

Copy-Item -Force (Join-Path $outDir 'libohos.h') (Join-Path $rootDir 'entry\src\main\cpp\include\libohos.h')

Write-Output "Built $(Join-Path $outDir 'libohos.so')"

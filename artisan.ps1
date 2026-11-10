$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    if ($args.Count -eq 0) { go run -buildvcs=false . artisan help }
    else { go run -buildvcs=false . artisan @args }
    exit $LASTEXITCODE
} finally { Pop-Location }


$BaseUrl = "http://localhost:8080"
$MaxRetries = 30
$RetryDelay = 2

Write-Host "Waiting for service to start..." -ForegroundColor Yellow

$retries = 0
$serviceReady = $false

while ($retries -lt $MaxRetries -and -not $serviceReady) {
    try {
        $response = Invoke-WebRequest -Uri "$BaseUrl/health" -Method Get -TimeoutSec 2 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $serviceReady = $true
            Write-Host "[OK] Service is up and ready!" -ForegroundColor Green
        }
    } catch {
        $retries++
        Write-Host "Attempt $retries/$MaxRetries..." -ForegroundColor Gray
        Start-Sleep -Seconds $RetryDelay
    }
}

if (-not $serviceReady) {
    Write-Host "[ERROR] Failed to wait for service startup" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== Testing basic functionality ===" -ForegroundColor Cyan
Write-Host ""

Write-Host "Test 1: Create team..." -ForegroundColor Yellow
try {
    $teamBody = @{
        team_name = "test-team"
        members = @(
            @{user_id = "t1"; username = "TestUser1"; is_active = $true},
            @{user_id = "t2"; username = "TestUser2"; is_active = $true},
            @{user_id = "t3"; username = "TestUser3"; is_active = $true}
        )
    } | ConvertTo-Json -Depth 10

    $response = Invoke-RestMethod -Uri "$BaseUrl/team/add" -Method Post -Body $teamBody -ContentType "application/json"
    Write-Host "[OK] Team created successfully" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Failed to create team: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Test 2: Get team..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BaseUrl/team/get?team_name=test-team" -Method Get
    if ($response.team_name -eq "test-team" -and $response.members.Count -eq 3) {
        Write-Host "[OK] Team retrieved successfully" -ForegroundColor Green
    } else {
        Write-Host "[ERROR] Invalid team data" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "[ERROR] Failed to get team: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Test 3: Create Pull Request..." -ForegroundColor Yellow
try {
    $prBody = @{
        pull_request_id = "test-pr-001"
        pull_request_name = "Test PR"
        author_id = "t1"
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$BaseUrl/pullRequest/create" -Method Post -Body $prBody -ContentType "application/json"
    if ($response.pr.status -eq "OPEN" -and $response.pr.assigned_reviewers.Count -gt 0) {
        Write-Host "[OK] PR created successfully with $($response.pr.assigned_reviewers.Count) reviewer(s)" -ForegroundColor Green
    } else {
        Write-Host "[ERROR] PR created but with invalid data" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "[ERROR] Failed to create PR: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Test 4: Merge Pull Request..." -ForegroundColor Yellow
try {
    $mergeBody = @{
        pull_request_id = "test-pr-001"
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$BaseUrl/pullRequest/merge" -Method Post -Body $mergeBody -ContentType "application/json"
    if ($response.pr.status -eq "MERGED") {
        Write-Host "[OK] PR merged successfully" -ForegroundColor Green
    } else {
        Write-Host "[ERROR] PR was not merged" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "[ERROR] Failed to merge PR: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== All tests passed successfully! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Service is working correctly and ready to use." -ForegroundColor Cyan
Write-Host "API is available at: $BaseUrl" -ForegroundColor Cyan

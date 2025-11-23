# PowerShell скрипт для ручного тестирования API

$BaseUrl = "http://localhost:8080"

Write-Host "=== Testing PR Reviewer Assignment Service ===" -ForegroundColor Green
Write-Host ""

# 1. Health check
Write-Host "1. Health Check" -ForegroundColor Yellow
Invoke-RestMethod -Uri "$BaseUrl/health" -Method Get | ConvertTo-Json
Write-Host ""

# 2. Create team
Write-Host "2. Create Team 'backend'" -ForegroundColor Yellow
$teamBody = @{
    team_name = "backend"
    members = @(
        @{user_id = "u1"; username = "Alice"; is_active = $true},
        @{user_id = "u2"; username = "Bob"; is_active = $true},
        @{user_id = "u3"; username = "Charlie"; is_active = $true}
    )
} | ConvertTo-Json -Depth 10

Invoke-RestMethod -Uri "$BaseUrl/team/add" -Method Post -Body $teamBody -ContentType "application/json" | ConvertTo-Json -Depth 10
Write-Host ""

# 3. Get team
Write-Host "3. Get Team 'backend'" -ForegroundColor Yellow
Invoke-RestMethod -Uri "$BaseUrl/team/get?team_name=backend" -Method Get | ConvertTo-Json -Depth 10
Write-Host ""

# 4. Create PR
Write-Host "4. Create Pull Request" -ForegroundColor Yellow
$prBody = @{
    pull_request_id = "pr-1001"
    pull_request_name = "Add search feature"
    author_id = "u1"
} | ConvertTo-Json

Invoke-RestMethod -Uri "$BaseUrl/pullRequest/create" -Method Post -Body $prBody -ContentType "application/json" | ConvertTo-Json -Depth 10
Write-Host ""

# 5. Get user reviews
Write-Host "5. Get Reviews for user u2" -ForegroundColor Yellow
Invoke-RestMethod -Uri "$BaseUrl/users/getReview?user_id=u2" -Method Get | ConvertTo-Json -Depth 10
Write-Host ""

# 6. Deactivate user
Write-Host "6. Deactivate user u3" -ForegroundColor Yellow
$deactivateBody = @{
    user_id = "u3"
    is_active = $false
} | ConvertTo-Json

Invoke-RestMethod -Uri "$BaseUrl/users/setIsActive" -Method Post -Body $deactivateBody -ContentType "application/json" | ConvertTo-Json -Depth 10
Write-Host ""

# 7. Merge PR
Write-Host "7. Merge Pull Request" -ForegroundColor Yellow
$mergeBody = @{
    pull_request_id = "pr-1001"
} | ConvertTo-Json

Invoke-RestMethod -Uri "$BaseUrl/pullRequest/merge" -Method Post -Body $mergeBody -ContentType "application/json" | ConvertTo-Json -Depth 10
Write-Host ""

# 8. Try to reassign after merge (should fail)
Write-Host "8. Try to Reassign After Merge (should fail)" -ForegroundColor Yellow
$reassignBody = @{
    pull_request_id = "pr-1001"
    old_user_id = "u2"
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri "$BaseUrl/pullRequest/reassign" -Method Post -Body $reassignBody -ContentType "application/json" | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Expected error: $_" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== Testing Complete ===" -ForegroundColor Green

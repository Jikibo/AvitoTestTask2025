@echo off
REM

set BASE_URL=http://localhost:8080

echo === Testing PR Reviewer Assignment Service ===
echo.

echo 1. Health Check
curl -s %BASE_URL%/health
echo.
echo.

echo 2. Create Team 'backend'
curl -s -X POST %BASE_URL%/team/add ^
  -H "Content-Type: application/json" ^
  -d "{\"team_name\":\"backend\",\"members\":[{\"user_id\":\"u1\",\"username\":\"Alice\",\"is_active\":true},{\"user_id\":\"u2\",\"username\":\"Bob\",\"is_active\":true},{\"user_id\":\"u3\",\"username\":\"Charlie\",\"is_active\":true}]}"
echo.
echo.

echo 3. Get Team 'backend'
curl -s %BASE_URL%/team/get?team_name=backend
echo.
echo.

echo 4. Create Pull Request
curl -s -X POST %BASE_URL%/pullRequest/create ^
  -H "Content-Type: application/json" ^
  -d "{\"pull_request_id\":\"pr-1001\",\"pull_request_name\":\"Add search feature\",\"author_id\":\"u1\"}"
echo.
echo.

echo 5. Get Reviews for user u2
curl -s %BASE_URL%/users/getReview?user_id=u2
echo.
echo.

echo 6. Merge Pull Request
curl -s -X POST %BASE_URL%/pullRequest/merge ^
  -H "Content-Type: application/json" ^
  -d "{\"pull_request_id\":\"pr-1001\"}"
echo.
echo.

echo === Testing Complete ===
pause

#!/bin/bash
# Скрипт для ручного тестирования API

BASE_URL="http://localhost:8080"

echo "=== Testing PR Reviewer Assignment Service ==="
echo ""

# 1. Health check
echo "1. Health Check"
curl -s "$BASE_URL/health" | jq .
echo -e "\n"

# 2. Create team
echo "2. Create Team 'backend'"
curl -s -X POST "$BASE_URL/team/add" \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }' | jq .
echo -e "\n"

# 3. Get team
echo "3. Get Team 'backend'"
curl -s "$BASE_URL/team/get?team_name=backend" | jq .
echo -e "\n"

# 4. Create PR
echo "4. Create Pull Request"
curl -s -X POST "$BASE_URL/pullRequest/create" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1"
  }' | jq .
echo -e "\n"

# 5. Get user reviews
echo "5. Get Reviews for user u2"
curl -s "$BASE_URL/users/getReview?user_id=u2" | jq .
echo -e "\n"

# 6. Deactivate user
echo "6. Deactivate user u3"
curl -s -X POST "$BASE_URL/users/setIsActive" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "u3",
    "is_active": false
  }' | jq .
echo -e "\n"

# 7. Merge PR
echo "7. Merge Pull Request"
curl -s -X POST "$BASE_URL/pullRequest/merge" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }' | jq .
echo -e "\n"

# 8. Try to reassign after merge (should fail)
echo "8. Try to Reassign After Merge (should fail)"
curl -s -X POST "$BASE_URL/pullRequest/reassign" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u2"
  }' | jq .
echo -e "\n"

echo "=== Testing Complete ==="

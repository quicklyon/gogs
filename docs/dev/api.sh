#!/bin/bash

# b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2

# curl  -X 'GET' \
#   'http://localhost:3000/api/v1/repos/ysicing/111111/pulls?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json'

# curl  -X 'GET' \
#   'http://localhost:3000/api/v1/repos/ysicing/111111/pulls/1?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json'

# curl -X 'POST' \
#   'http://localhost:3000/api/v1/repos/ysicing/111111/pulls?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json' \
#   -H 'Content-Type: application/json' \
#   -d '{
#   "assignee": "",
#   "base": "master",
#   "body": "stringx",
#   "head": "feat/0.12.13",
#   "labels": [],
#   "milestone": 0,
#   "title": "stringx"
# }'

# curl -X 'PATCH' \
#   'http://localhost:3000/api/v1/repos/ysicing/111111/pulls/1?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json' \
#   -H 'Content-Type: application/json' \
#   -d '{
#   "assignee": "",
#   "body": "string",
#   "labels": [],
#   "milestone": 0,
#   "state": "open",
#   "title": "string"
# }'

# curl -X 'POST' \
#   'http://localhost:3000/api/v1/repos/ysicing/111111/pulls/6/merge?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json' \
#   -H 'Content-Type: application/json' \
#   -d '{"merge_style": "create_merge_commit", "commit_description": "string"}'

# curl -X 'GET' \
#   'http://localhost:3000/api/v1/admin/users?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json'

# curl -X 'GET' \
#   'http://localhost:3000/api/v1/admin/users?page=1&limit=1&token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json'

# curl -X 'GET' \
#   'http://localhost:3000/api/v1/repos/ysicing/111111/branch_protections?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
#   -H 'accept: application/json'

curl -X 'GET' \
  'http://localhost:3000/api/v1/repos/ysicing/111111/pulls/6.diff?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
  -H 'accept: text/plain'

curl -X 'GET' \
  'http://localhost:3000/api/v1/repos/ysicing/111111/pulls/6.patch?token=b6e1f2742824e354b6c0ae13510d0f8fe05fe8e2' \
  -H 'accept: text/plain'

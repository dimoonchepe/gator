-- name: CreateFeed :one

INSERT INTO feeds (id, name, url, user_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
RETURNING id;

-- name: GetAllFeeds :many

SELECT feeds.id, feeds.name, feeds.url, feeds.user_id, users.name as user_name, feeds.created_at, feeds.updated_at
FROM feeds
LEFT JOIN users ON feeds.user_id = users.id;

-- name: GetFeedByURL :one

SELECT feeds.id, feeds.name, feeds.url, feeds.user_id, users.name as user_name, feeds.created_at, feeds.updated_at
FROM feeds
LEFT JOIN users ON feeds.user_id = users.id
WHERE feeds.url = $1;

-- name: MarkFeedFetched :exec

UPDATE feeds SET last_fetched_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: GetNextFeedToFetch :one

SELECT feeds.id, feeds.name, feeds.url, feeds.user_id, feeds.created_at, feeds.updated_at
FROM feeds
ORDER BY feeds.last_fetched_at ASC NULLS FIRST
LIMIT 1;

-- name: CreateFeedFollow :one

WITH feed_follows AS (
    INSERT INTO feed_follows (user_id, feed_id)
    VALUES ($1, $2)
    RETURNING *
)

SELECT feed_follows.*, users.name as user_name, feeds.name as feed_name
FROM feed_follows
INNER JOIN users ON feed_follows.user_id = users.id
INNER JOIN feeds ON feed_follows.feed_id = feeds.id;

-- name: UnfollowFeed :exec

DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;

-- name: GetFeedFollowsForUser :many

SELECT feed_follows.*, users.name as user_name, feeds.name as feed_name
FROM feed_follows
INNER JOIN users ON feed_follows.user_id = users.id
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE feed_follows.user_id = $1;

-- name: CreatePost :exec
INSERT INTO posts ( id, title, url, description, feed_id, published_at )
VALUES ( $1, $2, $3, $4, $5, $6 );

-- name: GetPostsForUser :many
SELECT * FROM posts WHERE feed_id in (
    SELECT id FROM feeds WHERE user_id = $1)
ORDER BY published_at DESC;

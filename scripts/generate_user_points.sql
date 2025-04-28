-- Create a temporary table to store the results
CREATE TEMPORARY TABLE user_points (
    username VARCHAR(255),
    points INT
);

-- Insert user points data
INSERT INTO user_points (username, points)
WITH user_community_posts AS (
    -- Get posts per user per community (excluding own community)
    SELECT 
        u.username,
        p.community_id,
        COUNT(*) * 2 AS post_points
    FROM users u
    JOIN posts p ON u.id = p.user_id
    JOIN communities c ON p.community_id = c.id
    WHERE c.user_id != u.id  -- Exclude own community
    GROUP BY u.username, p.community_id
),
user_community_comments AS (
    -- Get comments per user per community (excluding own community)
    SELECT 
        u.username,
        c.community_id,
        COUNT(*) AS comment_points
    FROM users u
    JOIN comments c ON u.id = c.user_id
    JOIN communities cm ON c.community_id = cm.id
    WHERE cm.user_id != u.id  -- Exclude own community
    GROUP BY u.username, c.community_id
),
all_community_activities AS (
    -- Combine all activities using UNION to get all unique username/community combinations
    SELECT username, community_id FROM user_community_posts
    UNION
    SELECT username, community_id FROM user_community_comments
),
community_points AS (
    -- Calculate points for each community with 10-point cap
    SELECT 
        a.username,
        a.community_id,
        LEAST(
            COALESCE(p.post_points, 0) + 
            COALESCE(c.comment_points, 0),
            10
        ) AS community_total
    FROM all_community_activities a
    LEFT JOIN user_community_posts p ON a.username = p.username AND a.community_id = p.community_id
    LEFT JOIN user_community_comments c ON a.username = c.username AND a.community_id = c.community_id
)
-- Sum up points across all communities
SELECT 
    username,
    SUM(community_total) AS points
FROM community_points
GROUP BY username
ORDER BY points DESC;

-- Output results to console
SELECT * FROM user_points;

-- Clean up
DROP TEMPORARY TABLE user_points; 
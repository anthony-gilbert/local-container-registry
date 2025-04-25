-- Table for GitHub commits
CREATE TABLE IF NOT EXISTS github (
    commitSHA TEXT PRIMARY KEY,
    commitDescription TEXT NOT NULL
);

-- Table for Docker images
CREATE TABLE IF NOT EXISTS docker (
    imageID   TEXT PRIMARY KEY,
    imageSize BIGINT,
    imageTag  TEXT
);

-- “deployments” placeholder (PostgreSQL requires ≥1 column)
CREATE TABLE IF NOT EXISTS deployments (
    deployment_id SERIAL PRIMARY KEY
);

CREATE TABLE content_items (
    id BIGSERIAL PRIMARY KEY,
    public_path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_type TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (content_type IN ('course_article', 'article', 'geektime_article', 'love_course_article', 'pdf')),
    CHECK (LEFT(public_path, 1) = '/'),
    CHECK (length(public_path) <= 1024)
);

CREATE INDEX idx_content_items_active_path ON content_items (public_path) WHERE is_active;
CREATE INDEX idx_content_items_active_type ON content_items (content_type, public_path) WHERE is_active;

CREATE TABLE courses (
    id BIGSERIAL PRIMARY KEY,
    public_path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    article_count INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (public_path ~ '^/专栏/[^/]+$'),
    CHECK (length(public_path) <= 1024)
);

CREATE INDEX idx_courses_active_path ON courses (public_path) WHERE is_active;

CREATE TABLE course_articles (
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    content_item_id BIGINT NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    position INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, content_item_id),
    CHECK (position >= 0)
);

CREATE UNIQUE INDEX idx_course_articles_active_course_position
    ON course_articles (course_id, position) WHERE is_active;
CREATE INDEX idx_course_articles_active_content_item ON course_articles (content_item_id) WHERE is_active;

CREATE TABLE user_course_progress (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    learned_article_count INT NOT NULL DEFAULT 0,
    average_progress_percent INT NOT NULL DEFAULT 0,
    latest_content_item_id BIGINT REFERENCES content_items(id) ON DELETE SET NULL,
    latest_read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, course_id),
    CHECK (learned_article_count >= 0),
    CHECK (average_progress_percent >= 0 AND average_progress_percent <= 100)
);

CREATE INDEX idx_user_course_progress_user_latest ON user_course_progress (user_id, latest_read_at DESC);

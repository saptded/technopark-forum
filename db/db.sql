CREATE EXTENSION IF NOT EXISTS CITEXT;
DROP TABLE IF EXISTS users, forums, threads, posts, votes, forum_users CASCADE;

CREATE TABLE users
(
    email    CITEXT UNIQUE             NOT NULL,
    nickname CITEXT COLLATE "C" UNIQUE NOT NULL,
    fullname TEXT                      NOT NULL,
    about    TEXT DEFAULT NULL
);

CREATE TABLE forums
(
    title   VARCHAR            NOT NULL,
    author  CITEXT COLLATE "C" NOT NULL,
    slug    CITEXT PRIMARY KEY,
    posts   BIGINT             NOT NULL DEFAULT 0,
    threads INTEGER            NOT NULL DEFAULT 0
);

CREATE TABLE threads
(
    id         SERIAL PRIMARY KEY,
    title      TEXT   NOT NULL,
    author     CITEXT NOT NULL,
    forum      CITEXT NOT NULL,
    message    TEXT   NOT NULL,
    votes      INTEGER DEFAULT 0,
    slug       CITEXT,
    created_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE forum_users
(
    forum    CITEXT COLLATE "C",
    nickname CITEXT COLLATE "C",
    CONSTRAINT fk UNIQUE (forum, nickname)
);

CREATE TABLE posts
(
    id          SERIAL PRIMARY KEY,
    author      TEXT    NOT NULL,
    message     TEXT    NOT NULL,
    is_edited   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP WITH TIME ZONE,
    forum       TEXT    NOT NULL,
    thread      INTEGER NOT NULL,

    parent      INTEGER          DEFAULT 0,
    parents     INT[]   NOT NULL,
    main_parent INT     NOT NULL
);

CREATE TABLE votes
(
    id            SERIAL,
    user_nickname CITEXT  NOT NULL,
    thread_id     INTEGER NOT NULL REFERENCES threads,
    voice         INTEGER,
    prev_voice    INTEGER DEFAULT 0,
    CONSTRAINT unique_user_and_thread UNIQUE (user_nickname, thread_id)
);

CREATE UNIQUE INDEX threads_slug_idx ON threads (slug);
CREATE INDEX threads_slug_id_idx ON threads (slug, id);
CREATE INDEX threads_forum_created_at_idx ON threads (forum, created_at);
CREATE INDEX threads_forum_created_at_desc_idx ON threads (forum, created_at DESC);
CREATE UNIQUE INDEX threads_id_forum_idx ON threads (id, forum);
CREATE UNIQUE INDEX threads_slug_forum_idx ON threads (slug, forum);
CREATE UNIQUE INDEX threads_cover_idx
    ON threads (created_at, id, slug, title, message, forum, author, votes);
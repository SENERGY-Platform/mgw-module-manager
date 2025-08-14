CREATE TABLE IF NOT EXISTS modules
(
    id      VARCHAR(256)          NOT NULL,
    dir     VARCHAR(256)          NOT NULL,
    source  VARCHAR(512)          NOT NULL,
    channel VARCHAR(256)          NOT NULL,
    added   TIMESTAMP(6)          NOT NULL,
    updated TIMESTAMP(6)          NOT NULL,
    PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS modules
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    id      VARCHAR(256)          NOT NULL,
    dir     VARCHAR(256)          NOT NULL,
    source VARCHAR(512) NOT NULL,
    channel VARCHAR(256)          NOT NULL,
    added   TIMESTAMP(6)          NOT NULL,
    updated TIMESTAMP(6)          NOT NULL,
    UNIQUE KEY (id),
    PRIMARY KEY (`index`)
);
CREATE TABLE IF NOT EXISTS mod_dependencies
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    mod_id VARCHAR(256)          NOT NULL,
    req_id VARCHAR(256)          NOT NULL,
    UNIQUE KEY (mod_id, req_id),
    PRIMARY KEY (`index`),
    FOREIGN KEY (mod_id) REFERENCES modules (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
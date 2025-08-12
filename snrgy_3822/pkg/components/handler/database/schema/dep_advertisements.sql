CREATE TABLE IF NOT EXISTS dep_advertisements
(
    `index`     BIGINT AUTO_INCREMENT NOT NULL,
    id        CHAR(36)              NOT NULL,
    dep_id    CHAR(36)              NOT NULL,
    mod_id    VARCHAR(256)          NOT NULL,
    origin    VARCHAR(256)          NOT NULL,
    ref       VARCHAR(256)          NOT NULL,
    timestamp TIMESTAMP(6)          NOT NULL,
    UNIQUE KEY (id),
    UNIQUE KEY (dep_id, ref),
    PRIMARY KEY (`index`),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_adv_items
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    adv_id CHAR(36)              NOT NULL,
    `key`    VARCHAR(256)          NOT NULL,
    value  VARCHAR(512),
    UNIQUE KEY (adv_id, `key`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (adv_id) REFERENCES dep_advertisements (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
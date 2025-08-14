CREATE TABLE IF NOT EXISTS dep_advertisements
(
    id        CHAR(36)     NOT NULL,
    dep_id    CHAR(36)     NOT NULL,
    mod_id    VARCHAR(256) NOT NULL,
    origin    VARCHAR(256) NOT NULL,
    ref       VARCHAR(256) NOT NULL,
    timestamp TIMESTAMP(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY (dep_id, ref),
    INDEX (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_adv_items
(
    dep_adv_id CHAR(36)     NOT NULL,
    item_key   VARCHAR(256) NOT NULL,
    item_value VARCHAR(512),
    UNIQUE KEY (dep_adv_id, item_key),
    INDEX (dep_adv_id),
    FOREIGN KEY (dep_adv_id) REFERENCES dep_advertisements (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
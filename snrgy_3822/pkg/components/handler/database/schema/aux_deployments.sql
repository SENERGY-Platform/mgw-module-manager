CREATE TABLE IF NOT EXISTS aux_deployments
(
    `index`      BIGINT AUTO_INCREMENT NOT NULL,
    id         CHAR(36)              NOT NULL,
    dep_id     CHAR(36)              NOT NULL,
    image      VARCHAR(256)          NOT NULL,
    created    TIMESTAMP(6)          NOT NULL,
    updated    TIMESTAMP(6)          NOT NULL,
    ref        VARCHAR(256)          NOT NULL,
    name       VARCHAR(256)          NOT NULL,
    enabled    BOOLEAN               NOT NULL,
    command    VARCHAR(512),
    pseudo_tty BOOLEAN,
    UNIQUE KEY (id),
    PRIMARY KEY (`index`),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_labels
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    aux_id CHAR(36)              NOT NULL,
    name   VARCHAR(256)          NOT NULL,
    value  VARCHAR(512),
    UNIQUE KEY (aux_id, name),
    PRIMARY KEY (`index`),
    FOREIGN KEY (aux_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_configs
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    aux_id CHAR(36)              NOT NULL,
    ref    VARCHAR(256)          NOT NULL,
    value  VARCHAR(512),
    UNIQUE KEY (aux_id, ref),
    PRIMARY KEY (`index`),
    FOREIGN KEY (aux_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_containers
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    aux_id CHAR(36)              NOT NULL,
    ctr_id VARCHAR(256)          NOT NULL,
    alias  VARCHAR(256)          NOT NULL,
    UNIQUE KEY (aux_id, ctr_id),
    PRIMARY KEY (`index`),
    FOREIGN KEY (aux_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_volumes
(
    `index`     BIGINT AUTO_INCREMENT NOT NULL,
    aux_id    CHAR(36)              NOT NULL,
    name      VARCHAR(256)          NOT NULL,
    mnt_point VARCHAR(256)          NOT NULL,
    UNIQUE KEY (aux_id, name),
    PRIMARY KEY (`index`),
    FOREIGN KEY (aux_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
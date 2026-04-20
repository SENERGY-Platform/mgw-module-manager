CREATE TABLE IF NOT EXISTS aux_deployments
(
    id         CHAR(36)     NOT NULL,
    dep_id     CHAR(36)     NOT NULL,
    image      VARCHAR(256) NOT NULL,
    created    TIMESTAMP(6) NOT NULL,
    updated    TIMESTAMP(6) NOT NULL,
    ref        VARCHAR(256) NOT NULL,
    name       VARCHAR(256) NOT NULL,
    enabled    BOOLEAN      NOT NULL,
    ctr_name   VARCHAR(256) NOT NULL,
    ctr_alias  VARCHAR(256) NOT NULL,
    recreate   BOOLEAN      NOT NULL,
    command    VARCHAR(512),
    pseudo_tty BOOLEAN,
    PRIMARY KEY (id),
    INDEX i_dep_id (dep_id),
    INDEX i_dep_id_ref (dep_id, ref),
    FOREIGN KEY fk_dep_id (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_dep_labels
(
    aux_dep_id CHAR(36)     NOT NULL,
    name       VARCHAR(256) NOT NULL,
    value      VARCHAR(512),
    UNIQUE KEY uk_aux_dep_id_name (aux_dep_id, name),
    INDEX i_aux_dep_id (aux_dep_id),
    FOREIGN KEY (aux_dep_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_dep_configs
(
    aux_dep_id CHAR(36)     NOT NULL,
    name       VARCHAR(256) NOT NULL,
    value      VARCHAR(512),
    UNIQUE KEY uk_aux_dep_id_name (aux_dep_id, name),
    INDEX i_aux_dep_id (aux_dep_id),
    FOREIGN KEY (aux_dep_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_dep_volumes
(
    id     VARCHAR(512) NOT NULL,
    dep_id CHAR(36)     NOT NULL,
    ref    VARCHAR(256) NOT NULL,
    name   VARCHAR(256) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_dep_volume_mounts
(
    vol_id     VARCHAR(512) NOT NULL,
    aux_dep_id CHAR(36)     NOT NULL,
    mnt_path   VARCHAR(512) NOT NULL,
    UNIQUE KEY uk_aux_dep_id_mnt_path (aux_dep_id, mnt_path),
    INDEX i_aux_dep_id (aux_dep_id),
    FOREIGN KEY (vol_id) REFERENCES aux_dep_volumes (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    FOREIGN KEY (aux_dep_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
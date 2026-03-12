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
    ref        VARCHAR(256) NOT NULL,
    value      VARCHAR(512),
    UNIQUE KEY uk_aux_dep_id_ref (aux_dep_id, ref),
    INDEX i_aux_dep_id (aux_dep_id),
    FOREIGN KEY (aux_dep_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_dep_containers
(
    aux_dep_id CHAR(36)     NOT NULL,
    ctr_id     VARCHAR(256) NOT NULL,
    alias      VARCHAR(256) NOT NULL,
    UNIQUE KEY uk_aux_dep_id_ctr_id (aux_dep_id, ctr_id),
    INDEX i_aux_dep_id (aux_dep_id),
    FOREIGN KEY (aux_dep_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS aux_dep_volumes
(
    aux_dep_id CHAR(36)     NOT NULL,
    name       VARCHAR(256) NOT NULL,
    mnt_point  VARCHAR(256) NOT NULL,
    UNIQUE KEY uk_aux_dep_id_name (aux_dep_id, name),
    INDEX i_aux_dep_id (aux_dep_id),
    FOREIGN KEY (aux_dep_id) REFERENCES aux_deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
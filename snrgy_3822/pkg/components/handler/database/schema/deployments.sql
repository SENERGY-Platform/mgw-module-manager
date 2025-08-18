CREATE TABLE IF NOT EXISTS deployments
(
    id       CHAR(36)     NOT NULL,
    mod_id   VARCHAR(256) NOT NULL,
    mod_ver  VARCHAR(256) NOT NULL,
    name     VARCHAR(256) NOT NULL,
    dir      VARCHAR(256) NOT NULL,
    enabled  BOOLEAN      NOT NULL,
    indirect BOOLEAN      NOT NULL,
    created  TIMESTAMP(6) NOT NULL,
    updated  TIMESTAMP(6) NOT NULL,
    PRIMARY KEY (id),
    INDEX i_mod_id (mod_id),
    FOREIGN KEY (mod_id) REFERENCES modules (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_containers
(
    dep_id  CHAR(36)     NOT NULL,
    ctr_id  VARCHAR(256) NOT NULL,
    srv_ref VARCHAR(256) NOT NULL,
    alias   VARCHAR(256) NOT NULL,
    `order` BIGINT       NOT NULL,
    UNIQUE KEY uk_dep_id_ctr_id_srv_ref (dep_id, ctr_id, srv_ref),
    INDEX i_dep_id (dep_id),
    INDEX i_ctr_id (ctr_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_host_resources
(
    dep_id CHAR(36)     NOT NULL,
    ref    VARCHAR(128) NOT NULL,
    res_id VARCHAR(256) NOT NULL,
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_secrets
(
    dep_id   CHAR(36)     NOT NULL,
    ref      VARCHAR(128) NOT NULL,
    sec_id   VARCHAR(256) NOT NULL,
    item     VARCHAR(128) NULL,
    as_mount BOOLEAN,
    as_env   BOOLEAN,
    UNIQUE KEY uk_dep_id_ref_item (dep_id, ref, item),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_configs
(
    dep_id   CHAR(36)     NOT NULL,
    ref      VARCHAR(128) NOT NULL,
    v_string VARCHAR(512),
    v_int    BIGINT,
    v_float  DOUBLE,
    v_bool   BOOLEAN,
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_list_configs
(
    dep_id   CHAR(36)     NOT NULL,
    ref      VARCHAR(128) NOT NULL,
    ord      SMALLINT     NOT NULL,
    v_string VARCHAR(512),
    v_int    BIGINT,
    v_float  DOUBLE,
    v_bool   BOOLEAN,
    UNIQUE KEY uk_dep_id_ref_ord (dep_id, ref, ord),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
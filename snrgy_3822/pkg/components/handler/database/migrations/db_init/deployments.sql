CREATE TABLE IF NOT EXISTS deployments
(
    id          CHAR(36)     NOT NULL,
    mod_id      VARCHAR(256) NOT NULL,
    mod_source  VARCHAR(512) NOT NULL,
    mod_channel VARCHAR(256) NOT NULL,
    mod_ver     VARCHAR(256) NOT NULL,
    name        VARCHAR(256) NOT NULL,
    dir         VARCHAR(256) NOT NULL,
    enabled     BOOLEAN      NOT NULL,
    created     TIMESTAMP(6) NOT NULL,
    updated     TIMESTAMP(6) NOT NULL,
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
CREATE TABLE IF NOT EXISTS dep_volumes
(
    dep_id CHAR(36)     NOT NULL,
    ref    VARCHAR(128) NOT NULL,
    name   VARCHAR(256) NOT NULL,
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
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
    item     VARCHAR(128) NULL, # e.g. user credentials consist of 'username' and 'password' stored as a single secret
    as_mount BOOLEAN,
    as_env   BOOLEAN,
    UNIQUE KEY uk_dep_id_ref_item (dep_id, ref, item),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_configs
(
    id        VARCHAR(256) NOT NULL,
    dep_id    CHAR(36)     NOT NULL,
    ref       VARCHAR(128) NOT NULL,
    data_type SMALLINT     NOT NULL,
    is_list   BOOLEAN      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_config_values
(
    c_id     VARCHAR(256) NOT NULL,
    v_string VARCHAR(512),
    v_int    BIGINT,
    v_float  DOUBLE,
    v_bool   BOOLEAN,
    ord      SMALLINT     NOT NULL,
    UNIQUE KEY uk_c_id_ord (c_id, ord),
    INDEX i_c_id (c_id),
    FOREIGN KEY (c_id) REFERENCES dep_configs (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_global_configs
(
    dep_id CHAR(36)     NOT NULL,
    ref    VARCHAR(128) NOT NULL,
    c_id   VARCHAR(256) NOT NULL,
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    FOREIGN KEY (c_id) REFERENCES global_configs (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_files
(
    dep_id CHAR(36)     NOT NULL,
    ref    VARCHAR(128) NOT NULL,
    data   LONGBLOB,
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_file_groups
(
    id     VARCHAR(256) NOT NULL,
    dep_id CHAR(36)     NOT NULL,
    ref    VARCHAR(128) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_dep_id_ref (dep_id, ref),
    INDEX i_dep_id (dep_id),
    FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS dep_file_group_files
(
    g_id   VARCHAR(256) NOT NULL,
    path   VARCHAR(512) NOT NULL,
    format SMALLINT     NOT NULL,
    data   LONGBLOB,
    UNIQUE KEY uk_g_id_path (g_id, path),
    INDEX i_g_id (g_id),
    FOREIGN KEY (g_id) REFERENCES dep_file_groups (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
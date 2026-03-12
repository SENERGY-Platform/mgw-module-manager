CREATE TABLE IF NOT EXISTS global_configs
(
    id        CHAR(36)     NOT NULL,
    name      VARCHAR(256) NOT NULL,
    data_type SMALLINT     NOT NULL,
    is_list   BOOLEAN      NOT NULL,
    PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS global_config_values
(
    c_id     CHAR(36) NOT NULL,
    v_string VARCHAR(512),
    v_int    BIGINT,
    v_float  DOUBLE,
    v_bool   BOOLEAN,
    ord      SMALLINT NOT NULL,
    UNIQUE KEY uk_c_id_ord (c_id, ord),
    INDEX i_id (c_id),
    FOREIGN KEY (c_id) REFERENCES global_configs (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
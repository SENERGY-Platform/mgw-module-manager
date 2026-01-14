CREATE TABLE IF NOT EXISTS global_configs
(
    id   CHAR(36)     NOT NULL,
    name VARCHAR(256) NOT NULL,
    is_list BOOLEAN NOT NULL,
    v_string VARCHAR(512),
    v_int    BIGINT,
    v_float  DOUBLE,
    v_bool   BOOLEAN,
    ord      SMALLINT     NOT NULL,
    UNIQUE KEY uk_id_ord (id, ord),
    INDEX i_id (id)
);
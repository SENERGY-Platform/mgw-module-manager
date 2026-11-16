CREATE TABLE IF NOT EXISTS auto_update
(
    id            TINYINT     NOT NULL,
    enabled       BOOLEAN     NOT NULL,
    level         VARCHAR(16) NOT NULL,
    window_start  CHAR(5)     NOT NULL,
    window_end    CHAR(5)     NOT NULL,
    time_zone     VARCHAR(64) NOT NULL,
    last_run_date CHAR(10),
    last_run      JSON,
    PRIMARY KEY (id)
);

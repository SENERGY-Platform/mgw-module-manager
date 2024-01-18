CREATE TABLE IF NOT EXISTS `modules`
(
    `index`    BIGINT AUTO_INCREMENT NOT NULL,
    `id`       VARCHAR(256)          NOT NULL,
    `dir`      VARCHAR(256)          NOT NULL,
    `modfile`  VARCHAR(16)           NOT NULL,
    `added`    TIMESTAMP(6)          NOT NULL,
    `updated`  TIMESTAMP(6)          NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`)
);
CREATE TABLE IF NOT EXISTS `mod_dependencies`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `mod_id` VARCHAR(256)              NOT NULL,
    `req_id` VARCHAR(256)              NOT NULL,
    UNIQUE KEY (`mod_id`, `req_id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`mod_id`) REFERENCES `modules` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `deployments`
(
    `index`    BIGINT AUTO_INCREMENT NOT NULL,
    `id`       CHAR(36)              NOT NULL,
    `mod_id`   VARCHAR(256)          NOT NULL,
    `mod_ver`  VARCHAR(256)          NOT NULL,
    `name`     VARCHAR(256)          NOT NULL,
    `dir`      VARCHAR(256)          NOT NULL,
    `enabled`  BOOLEAN               NOT NULL,
    `indirect` BOOLEAN               NOT NULL,
    `created`  TIMESTAMP(6)          NOT NULL,
    `updated`  TIMESTAMP(6)          NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`mod_id`) REFERENCES `modules` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `dependencies`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)              NOT NULL,
    `req_id` CHAR(36)              NOT NULL,
    UNIQUE KEY (`dep_id`, `req_id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `containers`
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id`  CHAR(36)              NOT NULL,
    `ctr_id`  VARCHAR(256)          NOT NULL,
    `srv_ref` VARCHAR(256)          NOT NULL,
    `alias`   VARCHAR(256)          NOT NULL,
    `order`   BIGINT                NOT NULL,
    UNIQUE KEY (`dep_id`, `ctr_id`, `srv_ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `host_resources`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)              NOT NULL,
    `ref`    VARCHAR(128)          NOT NULL,
    `res_id` VARCHAR(256)          NOT NULL,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `secrets`
(
    `index`    BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id`   CHAR(36)              NOT NULL,
    `ref`      VARCHAR(128)          NOT NULL,
    `sec_id`   VARCHAR(256)          NOT NULL,
    `item`     VARCHAR(128)          NULL,
    `as_mount` BOOLEAN,
    `as_env`   BOOLEAN,
    UNIQUE KEY (`dep_id`, `ref`, `item`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs`
(
    `index`    BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id`   CHAR(36)              NOT NULL,
    `ref`      VARCHAR(128)          NOT NULL,
    `v_string` VARCHAR(512),
    `v_int`    BIGINT,
    `v_float`  DOUBLE,
    `v_bool`   BOOLEAN,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `list_configs`
(
    `index`    BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id`   CHAR(36)              NOT NULL,
    `ref`      VARCHAR(128)          NOT NULL,
    `ord`      SMALLINT              NOT NULL,
    `v_string` VARCHAR(512),
    `v_int`    BIGINT,
    `v_float`  DOUBLE,
    `v_bool`   BOOLEAN,
    UNIQUE KEY (`dep_id`, `ref`, `ord`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `aux_deployments`
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    `id`      CHAR(36)              NOT NULL,
    `dep_id`  CHAR(36)              NOT NULL,
    `image`   VARCHAR(256)          NOT NULL,
    `created` TIMESTAMP(6)          NOT NULL,
    `updated` TIMESTAMP(6)          NOT NULL,
    `ref`     VARCHAR(256)          NOT NULL,
    `name`    VARCHAR(256)          NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `aux_labels`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `aux_id` CHAR(36)              NOT NULL,
    `name`   VARCHAR(256)          NOT NULL,
    `value`  VARCHAR(512),
    UNIQUE KEY (aux_id, `name`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (aux_id) REFERENCES `aux_deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `aux_configs`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `aux_id` CHAR(36)              NOT NULL,
    `ref`    VARCHAR(256)          NOT NULL,
    `value`  VARCHAR(512),
    UNIQUE KEY (`aux_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`aux_id`) REFERENCES `aux_deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `aux_containers`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `aux_id` CHAR(36)              NOT NULL,
    `ctr_id` VARCHAR(256)          NOT NULL,
    `alias`  VARCHAR(256)          NOT NULL,
    UNIQUE KEY (`aux_id`, `ctr_id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`aux_id`) REFERENCES `aux_deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
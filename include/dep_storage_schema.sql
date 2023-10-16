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
    PRIMARY KEY (`index`)
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
CREATE TABLE IF NOT EXISTS `instances`
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    `id`      CHAR(36)              NOT NULL,
    `dep_id`  CHAR(36)              NOT NULL,
    `created` TIMESTAMP(6)          NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `inst_containers`
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    `inst_id` CHAR(36)              NOT NULL,
    `srv_ref` VARCHAR(256)          NOT NULL,
    `order`   BIGINT                NOT NULL,
    `ctr_id`  VARCHAR(256)          NOT NULL,
    UNIQUE KEY (`ctr_id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`inst_id`) REFERENCES `instances` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
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
CREATE TABLE IF NOT EXISTS `sub_deployments`
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    `id`      CHAR(36)              NOT NULL,
    `dep_id`  CHAR(36)              NOT NULL,
    `image`   VARCHAR(256)          NOT NULL,
    `ctr_id`  VARCHAR(256)          NOT NULL,
    `created` TIMESTAMP(6)          NOT NULL,
    `updated` TIMESTAMP(6)          NOT NULL,
    `name`    VARCHAR(256)          NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `sd_labels`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `sd_id` CHAR(36)              NOT NULL,
    `key`   VARCHAR(256)          NOT NULL,
    `value` VARCHAR(512),
    UNIQUE KEY (`sd_id`, `key`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`sd_id`) REFERENCES `sub_deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `sd_configs`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `sd_id` CHAR(36)              NOT NULL,
    `ref`   VARCHAR(256)          NOT NULL,
    `value` VARCHAR(512),
    UNIQUE KEY (`sd_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`sd_id`) REFERENCES `sub_deployments` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
);
/*
 * Copyright 2023 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

CREATE TABLE IF NOT EXISTS `deployments`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `id`        CHAR(36)     NOT NULL,
    `module_id` VARCHAR(256) NOT NULL,
    `name`      VARCHAR(256) NOT NULL,
    `created`   TIMESTAMP    NOT NULL,
    `updated`   TIMESTAMP    NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`)
);
CREATE TABLE IF NOT EXISTS `instances`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `id`     CHAR(36) NOT NULL,
    `dep_id` CHAR(36) NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `containers`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `c_id` CHAR(36) NOT NULL,
    `i_id` CHAR(36) NOT NULL,
    UNIQUE KEY (`c_id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`i_id`) REFERENCES `instances` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `host_resources`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `res_id` CHAR(36)     NOT NULL,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `secrets`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `sec_id` CHAR(36)     NOT NULL,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_string`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `value`  VARCHAR(512),
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_string_list`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `ord`    SMALLINT     NOT NULL,
    `value`  VARCHAR(512),
    UNIQUE KEY (`dep_id`, `ref`, `ord`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_int`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `value`  BIGINT,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_int_list`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `ord`    SMALLINT     NOT NULL,
    `value`  BIGINT,
    UNIQUE KEY (`dep_id`, `ref`, `ord`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_float`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `value`  DOUBLE,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_float_list`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `ord`    SMALLINT     NOT NULL,
    `value`  DOUBLE,
    UNIQUE KEY (`dep_id`, `ref`, `ord`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_bool`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `value`  BOOLEAN,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `configs_bool_list`
(
    `index` BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)     NOT NULL,
    `ref`    VARCHAR(128) NOT NULL,
    `ord`    SMALLINT     NOT NULL,
    `value`  BOOLEAN,
    UNIQUE KEY (`dep_id`, `ref`, `ord`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
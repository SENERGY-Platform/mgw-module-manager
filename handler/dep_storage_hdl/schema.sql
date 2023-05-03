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
    `index`     BIGINT AUTO_INCREMENT NOT NULL,
    `id`        CHAR(36)              NOT NULL,
    `module_id` VARCHAR(256)          NOT NULL,
    `name`      VARCHAR(256)          NOT NULL,
    `stopped`   BOOLEAN               NOT NULL,
    `indirect`  BOOLEAN               NOT NULL,
    `created`   TIMESTAMP(6)          NOT NULL,
    `updated`   TIMESTAMP(6)          NOT NULL,
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
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `instances`
(
    `index`   BIGINT AUTO_INCREMENT NOT NULL,
    `id`      CHAR(36)              NOT NULL,
    `dep_id`  CHAR(36)              NOT NULL,
    `created` TIMESTAMP(6)          NOT NULL,
    `updated` TIMESTAMP(6)          NOT NULL,
    UNIQUE KEY (`id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
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
    FOREIGN KEY (`inst_id`) REFERENCES `instances` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `sub_containers`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)              NOT NULL,
    `grp_id` VARCHAR(256)          NOT NULL,
    `ref`    VARCHAR(256)          NOT NULL,
    `order`  BIGINT                NOT NULL,
    `ctr_id` VARCHAR(256)          NOT NULL,
    UNIQUE KEY (`ctr_id`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `host_resources`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)              NOT NULL,
    `ref`    VARCHAR(128)          NOT NULL,
    `res_id` CHAR(36)              NOT NULL,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
CREATE TABLE IF NOT EXISTS `secrets`
(
    `index`  BIGINT AUTO_INCREMENT NOT NULL,
    `dep_id` CHAR(36)              NOT NULL,
    `ref`    VARCHAR(128)          NOT NULL,
    `sec_id` CHAR(36)              NOT NULL,
    UNIQUE KEY (`dep_id`, `ref`),
    PRIMARY KEY (`index`),
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
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
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
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
    FOREIGN KEY (`dep_id`) REFERENCES `deployments` (`id`)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);
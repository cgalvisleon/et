-- ========== 2. Definición de modelos (DDL) · Define (declarativo) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_products` (
  `id` VARCHAR(80) NOT NULL,
  `name` VARCHAR(255) DEFAULT NULL,
  `category` VARCHAR(80) DEFAULT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_products_category_idx` ON `jsql_catalog`.`f_products` (`category`);

-- ========== 2. Definición de modelos (DDL) · DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_roles` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `name` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_roles_status_idx` ON `jsql_catalog`.`f_roles` (`status`);
CREATE INDEX `f_roles__idx_idx` ON `jsql_catalog`.`f_roles` (`_idx`);

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_users` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `name` VARCHAR(80) DEFAULT NULL,
  `email` VARCHAR(255) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE UNIQUE INDEX `f_users_email_key` ON `jsql_catalog`.`f_users` (`email`);
CREATE INDEX `f_users_status_idx` ON `jsql_catalog`.`f_users` (`status`);
CREATE INDEX `f_users__idx_idx` ON `jsql_catalog`.`f_users` (`_idx`);

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_users_f_roles` (
  `user_id` VARCHAR(80) NOT NULL,
  `role_id` VARCHAR(80) NOT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`user_id`, `role_id`)
);
CREATE INDEX `f_users_f_roles__idx_idx` ON `jsql_catalog`.`f_users_f_roles` (`_idx`);
ALTER TABLE `jsql_catalog`.`f_users_f_roles` ADD CONSTRAINT `fk_f_users_f_roles_f_users` FOREIGN KEY (`user_id`) REFERENCES `jsql_catalog`.`f_users` (`id`) ON DELETE CASCADE;
ALTER TABLE `jsql_catalog`.`f_users_f_roles` ADD CONSTRAINT `fk_f_users_f_roles_f_roles` FOREIGN KEY (`role_id`) REFERENCES `jsql_catalog`.`f_roles` (`id`) ON DELETE CASCADE;

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_doc_types` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `title` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_doc_types_status_idx` ON `jsql_catalog`.`f_doc_types` (`status`);
CREATE INDEX `f_doc_types__idx_idx` ON `jsql_catalog`.`f_doc_types` (`_idx`);

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_orders` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `user_id` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_orders_status_idx` ON `jsql_catalog`.`f_orders` (`status`);
CREATE INDEX `f_orders__idx_idx` ON `jsql_catalog`.`f_orders` (`_idx`);

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_orders_items` (
  `order_id` VARCHAR(80) DEFAULT NULL,
  `id` VARCHAR(80) NOT NULL,
  `product` VARCHAR(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
ALTER TABLE `jsql_catalog`.`f_orders_items` ADD CONSTRAINT `fk_f_orders_items_f_orders` FOREIGN KEY (`order_id`) REFERENCES `jsql_catalog`.`f_orders` (`id`) ON DELETE CASCADE;

-- ========== 2. Definición de modelos (DDL) · DefineTenantModel / DefineProjectModel [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_tenant` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `tenant_id` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_tenant_status_idx` ON `jsql_catalog`.`f_tenant` (`status`);
CREATE INDEX `f_tenant__idx_idx` ON `jsql_catalog`.`f_tenant` (`_idx`);
CREATE INDEX `f_tenant_tenant_id_idx` ON `jsql_catalog`.`f_tenant` (`tenant_id`);

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_project` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `project_id` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_project_status_idx` ON `jsql_catalog`.`f_project` (`status`);
CREATE INDEX `f_project__idx_idx` ON `jsql_catalog`.`f_project` (`_idx`);
CREATE INDEX `f_project_project_id_idx` ON `jsql_catalog`.`f_project` (`project_id`);

-- ========== 2. Definición de modelos (DDL) · DefineRequired (rechaza el insert sin el campo) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_required` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `name` VARCHAR(255) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_required_status_idx` ON `jsql_catalog`.`f_required` (`status`);
CREATE INDEX `f_required__idx_idx` ON `jsql_catalog`.`f_required` (`_idx`);

-- ========== 2. Definición de modelos (DDL) · DefineForeignKeys (rechaza un hijo sin padre) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_parent` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_parent_status_idx` ON `jsql_catalog`.`f_parent` (`status`);
CREATE INDEX `f_parent__idx_idx` ON `jsql_catalog`.`f_parent` (`_idx`);

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_child` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `parent_id` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_child_status_idx` ON `jsql_catalog`.`f_child` (`status`);
CREATE INDEX `f_child__idx_idx` ON `jsql_catalog`.`f_child` (`_idx`);
ALTER TABLE `jsql_catalog`.`f_child` ADD CONSTRAINT `fk_f_child_f_parent` FOREIGN KEY (`parent_id`) REFERENCES `jsql_catalog`.`f_parent` (`id`) ON DELETE CASCADE;

-- INSERT
INSERT INTO `jsql_catalog`.`f_parent`
  (`_idx`, `id`)
  VALUES ('1790553368528', 'p1');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`
) AS `result` FROM `jsql_catalog`.`f_parent` WHERE `id` = 'p1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_child`
  (`_idx`, `id`, `parent_id`)
  VALUES ('1790553368533', 'c1', 'p1');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."parent_id"', `parent_id`
) AS `result` FROM `jsql_catalog`.`f_child` WHERE `id` = 'c1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_child`
  (`_idx`, `id`, `parent_id`)
  VALUES ('1790553368536', 'c2', 'missing');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."parent_id"', `parent_id`
) AS `result` FROM `jsql_catalog`.`f_child` WHERE `id` = 'c2';

-- ========== 2. Definición de modelos (DDL) · Stricted (ignora campos desconocidos) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_strict` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `name` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_strict_status_idx` ON `jsql_catalog`.`f_strict` (`status`);
CREATE INDEX `f_strict__idx_idx` ON `jsql_catalog`.`f_strict` (`_idx`);

-- INSERT
INSERT INTO `jsql_catalog`.`f_strict`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368561', 's1', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_strict` WHERE `id` = 's1';

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_strict` AS A
WHERE A.`id` = 's1'
LIMIT 1

-- ========== 2. Definición de modelos (DDL) · DB.Query define (Define en JSON) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_json_model` (
  `id` VARCHAR(80) NOT NULL,
  `title` VARCHAR(255) DEFAULT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  PRIMARY KEY (`id`)
);

-- INSERT
INSERT INTO `jsql_catalog`.`f_json_model`
  (`id`, `title`, `_source`)
  VALUES ('j1', 'desde define', '{"extra":1}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."title"', `title`
) AS `result` FROM `jsql_catalog`.`f_json_model` WHERE `id` = 'j1';

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."title"', A.`title`
) AS `result`
FROM `jsql_catalog`.`f_json_model` AS A
WHERE A.`id` = 'j1'
LIMIT 1

-- ========== 2. Definición de modelos (DDL) · Insert (datos base) [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_doc_types`
  (`_idx`, `id`, `title`)
  VALUES ('1790553368577', 'CC', 'Cédula de ciudadanía');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."title"', `title`
) AS `result` FROM `jsql_catalog`.`f_doc_types` WHERE `id` = 'CC';

-- INSERT
INSERT INTO `jsql_catalog`.`f_doc_types`
  (`_idx`, `id`, `title`)
  VALUES ('1790553368579', 'NIT', 'Número de identificación tributaria');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."title"', `title`
) AS `result` FROM `jsql_catalog`.`f_doc_types` WHERE `id` = 'NIT';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'ana@example.com') THEN 'true' ELSE 'false' END AS `exists`

-- INSERT
INSERT INTO `jsql_catalog`.`f_users`
  (`_idx`, `email`, `id`, `name`, `_source`)
  VALUES ('1790553368582', 'ana@example.com', 'u1', 'Ana', '{"age":30,"password":"secret","tp_doc":"CC"}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`,
'$."email"', `email`
) AS `result` FROM `jsql_catalog`.`f_users` WHERE `id` = 'u1';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'luis@example.com') THEN 'true' ELSE 'false' END AS `exists`

-- INSERT
INSERT INTO `jsql_catalog`.`f_users`
  (`_idx`, `email`, `id`, `name`, `_source`)
  VALUES ('1790553368585', 'luis@example.com', 'u2', 'Luis', '{"age":17}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`,
'$."email"', `email`
) AS `result` FROM `jsql_catalog`.`f_users` WHERE `id` = 'u2';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'marta@example.com') THEN 'true' ELSE 'false' END AS `exists`

-- INSERT
INSERT INTO `jsql_catalog`.`f_users`
  (`_idx`, `email`, `id`, `name`, `_source`)
  VALUES ('1790553368587', 'marta@example.com', 'u3', 'Marta O''Neil', '{"age":45,"tp_doc":"NIT"}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`,
'$."email"', `email`
) AS `result` FROM `jsql_catalog`.`f_users` WHERE `id` = 'u3';

-- INSERT
INSERT INTO `jsql_catalog`.`f_roles`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368587', 'r1', 'admin');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_roles` WHERE `id` = 'r1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_roles`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368590', 'r2', 'editor');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_roles` WHERE `id` = 'r2';

-- INSERT
INSERT INTO `jsql_catalog`.`f_orders`
  (`_idx`, `id`, `user_id`, `_source`)
  VALUES ('1790553368591', 'o1', 'u1', '{"amount":100.5}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."user_id"', `user_id`
) AS `result` FROM `jsql_catalog`.`f_orders` WHERE `id` = 'o1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_orders`
  (`_idx`, `id`, `user_id`, `_source`)
  VALUES ('1790553368594', 'o2', 'u1', '{"amount":200}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."user_id"', `user_id`
) AS `result` FROM `jsql_catalog`.`f_orders` WHERE `id` = 'o2';

-- INSERT
INSERT INTO `jsql_catalog`.`f_orders`
  (`_idx`, `id`, `user_id`, `_source`)
  VALUES ('1790553368596', 'o3', 'u3', '{"amount":50}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."user_id"', `user_id`
) AS `result` FROM `jsql_catalog`.`f_orders` WHERE `id` = 'o3';

-- INSERT
INSERT INTO `jsql_catalog`.`f_orders_items`
  (`id`, `order_id`, `product`)
  VALUES ('i1', 'o1', 'router');
SELECT JSON_OBJECT('id', `id`, 'product', `product`) AS `result` FROM `jsql_catalog`.`f_orders_items` WHERE `id` = 'i1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_orders_items`
  (`id`, `order_id`, `product`)
  VALUES ('i2', 'o1', 'cable');
SELECT JSON_OBJECT('id', `id`, 'product', `product`) AS `result` FROM `jsql_catalog`.`f_orders_items` WHERE `id` = 'i2';

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('internet', 'p1', 'Plan 200', '{"price":90000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('internet', 'p2', 'Plan 500', '{"price":150000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p2';

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('tv', 'p3', 'Decoder', '{"price":20000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p3';

-- INSERT
INSERT INTO `jsql_catalog`.`f_users_f_roles`
  (`_idx`, `role_id`, `user_id`)
  VALUES ('1790553368605', 'r1', 'u1');
SELECT JSON_OBJECT('user_id', `user_id`, 'role_id', `role_id`) AS `result` FROM `jsql_catalog`.`f_users_f_roles` WHERE `user_id` = 'u1' AND `role_id` = 'r1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_users_f_roles`
  (`_idx`, `role_id`, `user_id`)
  VALUES ('1790553368607', 'r2', 'u1');
SELECT JSON_OBJECT('user_id', `user_id`, 'role_id', `role_id`) AS `result` FROM `jsql_catalog`.`f_users_f_roles` WHERE `user_id` = 'u1' AND `role_id` = 'r2';

-- INSERT
INSERT INTO `jsql_catalog`.`f_users_f_roles`
  (`_idx`, `role_id`, `user_id`)
  VALUES ('1790553368609', 'r2', 'u2');
SELECT JSON_OBJECT('user_id', `user_id`, 'role_id', `role_id`) AS `result` FROM `jsql_catalog`.`f_users_f_roles` WHERE `user_id` = 'u2' AND `role_id` = 'r2';

-- ========== 2. Definición de modelos (DDL) · DefineUnique (rechaza duplicados en insert, bulk y update) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'ana@example.com') THEN 'true' ELSE 'false' END AS `exists`

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'same@example.com') THEN 'true' ELSE 'false' END AS `exists`

-- BULK
INSERT INTO `jsql_catalog`.`f_users`
  (`_idx`, `email`, `id`, `name`)
  VALUES ('1790553368611', 'same@example.com', 'u8', 'Uno');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`,
'$."email"', `email`
) AS `result` FROM `jsql_catalog`.`f_users` WHERE `id` = 'u8';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'same@example.com') THEN 'true' ELSE 'false' END AS `exists`

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u2'
LIMIT 1000

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` = 'ana@example.com'
LIMIT 2

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u2'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_users`
  SET `created_at` = NULL,
    `email` = 'luis@example.com',
    `name` = 'Luis',
    `status` = 'active',
    `updated_at` = NULL,
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."age"', CAST('17' AS JSON)
)
  WHERE `id` = 'u2';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`,
'$."email"', `email`
) AS `result` FROM `jsql_catalog`.`f_users` WHERE `id` = 'u2';

-- QUERY
SELECT COUNT(*) AS `count`
FROM `jsql_catalog`.`f_users` AS A

-- ========== 3. Relaciones y campos calculados · Detail en el select (DefineDetail) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
WHERE A.`id` = 'o1'
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'product', A.`product`
) AS `result`
FROM `jsql_catalog`.`f_orders_items` AS A
WHERE A.`order_id` = 'o1'
LIMIT 30

-- ========== 3. Relaciones y campos calculados · Model.Detail + Query.Detail [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."user_id"', A.`user_id`
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
WHERE A.`id` = 'o1'
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'product', A.`product`
) AS `result`
FROM `jsql_catalog`.`f_orders_items` AS A
WHERE A.`order_id` = 'o1'
LIMIT 30

-- ========== 3. Relaciones y campos calculados · Master en el select (DefineMaster) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u1'
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'name', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_roles` AS A
INNER JOIN `jsql_catalog`.`f_users_f_roles` AS B
  ON B.`role_id` = A.`id`
WHERE B.`user_id` = 'u1'
LIMIT 1000

-- ========== 3. Relaciones y campos calculados · Master 1 a 1 (rows = 1) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u2'
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'name', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_roles` AS A
INNER JOIN `jsql_catalog`.`f_users_f_roles` AS B
  ON B.`role_id` = A.`id`
WHERE B.`user_id` = 'u2'
LIMIT 1

-- ========== 3. Relaciones y campos calculados · Model.Master + Model.Bridge [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_roles` AS A
INNER JOIN `jsql_catalog`.`f_users_f_roles` AS B
  ON A.`id` = B.`role_id`
WHERE B.`user_id` = 'u2'
LIMIT 1000

-- ========== 3. Relaciones y campos calculados · DefineRollup row (tp_doc → title) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'tp_doc', JSON_EXTRACT(A.`_source`, '$."tp_doc"')
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
ORDER BY A.`id` ASC
LIMIT 1000

-- QUERY
SELECT
JSON_OBJECT(
'title', A.`title`
) AS `result`
FROM `jsql_catalog`.`f_doc_types` AS A
WHERE A.`id` = 'CC'
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'title', A.`title`
) AS `result`
FROM `jsql_catalog`.`f_doc_types` AS A
WHERE A.`id` = NULL
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'title', A.`title`
) AS `result`
FROM `jsql_catalog`.`f_doc_types` AS A
WHERE A.`id` = 'NIT'
LIMIT 1

-- ========== 3. Relaciones y campos calculados · DefineRollup (count, sum, object) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u3'
LIMIT 1

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'amount', JSON_EXTRACT(A.`_source`, '$."amount"')
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
WHERE A.`user_id` = 'u3'
LIMIT 1

-- QUERY
SELECT COUNT(*) AS `count`
FROM `jsql_catalog`.`f_orders` AS A
WHERE A.`user_id` = 'u3'

-- QUERY
SELECT
JSON_OBJECT(
'amount', COALESCE(SUM(CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."amount"')) AS DECIMAL(38,10))), 0)
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
WHERE A.`user_id` = 'u3'
LIMIT 1000

-- ========== 3. Relaciones y campos calculados · DefineCalcFunc + Model.Calc [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u1'
LIMIT 1

-- ========== 3. Relaciones y campos calculados · Query.Calc con script JS (DefineCalc) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u3'
LIMIT 1

-- ========== 3. Relaciones y campos calculados · DefineCalc (script JS) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'name', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u3'
LIMIT 1

-- ========== 4. Consultas · Select + Where + OrderBy + All [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'name', A.`name`,
'age', JSON_EXTRACT(A.`_source`, '$."age"')
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) > 18
ORDER BY CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) DESC
LIMIT 1000

-- ========== 4. Consultas · One / First / Count / Exists [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u2'
LIMIT 1

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
ORDER BY A.`id` ASC
LIMIT 2

-- QUERY
SELECT COUNT(*) AS `count`
FROM `jsql_catalog`.`f_users` AS A

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u9') THEN 'true' ELSE 'false' END AS `exists`

-- ========== 4. Consultas · Limit (paginación) / Page [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
ORDER BY A.`id` ASC
LIMIT 1 OFFSET 1

-- ========== 4. Consultas · Hidden (campo oculto en la consulta y en el modelo) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."email"', '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u1'
LIMIT 1

-- ========== 4. Consultas · Hidden pedido por nombre (se devuelve) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'password', JSON_EXTRACT(A.`_source`, '$."password"')
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u1'
LIMIT 1

-- ========== 4. Consultas · Join (fluido) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'user', U.`name`
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
INNER JOIN `jsql_catalog`.`f_users` AS U
  ON A.`user_id` = U.`id`
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 4. Consultas · LeftJoin + GroupBy [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', ANY_VALUE(A.`id`),
'n', COUNT(O.`id`)
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
LEFT JOIN `jsql_catalog`.`f_orders` AS O
  ON O.`user_id` = A.`id`
GROUP BY A.`id`
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 4. Consultas · RightJoin / FullJoin [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', ANY_VALUE(U.`id`),
'n', COUNT(A.`id`)
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
RIGHT JOIN `jsql_catalog`.`f_users` AS U
  ON A.`user_id` = U.`id`
GROUP BY U.`id`
ORDER BY U.`id` ASC
LIMIT 1000

-- ========== 4. Consultas · GroupBy + Having (fluido) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'user_id', ANY_VALUE(A.`user_id`),
'n', COUNT(A.`id`),
'total', COALESCE(SUM(CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."amount"')) AS DECIMAL(38,10))), 0)
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
GROUP BY A.`user_id`
HAVING COUNT(A.`id`) > 1
LIMIT 1000

-- ========== 4. Consultas · OrderBy por alias de un agregado [pass]

-- QUERY
SELECT
JSON_OBJECT(
'user_id', ANY_VALUE(A.`user_id`),
'n', COUNT(A.`id`)
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
GROUP BY A.`user_id`
ORDER BY COUNT(A.`id`) DESC
LIMIT 1000

-- ========== 4. Consultas · Model.Query (JSON) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`,
'name', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) >= 18
ORDER BY A.`name` DESC
LIMIT 10

-- ========== 4. Consultas · Model.Query con from (otro modelo y alias) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', U.`id`,
'name', U.`name`
) AS `result`
FROM `jsql_catalog`.`f_users` AS U
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(U.`_source`, '$."age"')) AS SIGNED) > 18
ORDER BY U.`id` ASC
LIMIT 1000

-- ========== 4. Consultas · Model.Query con from de varios orígenes [pass]

-- QUERY
SELECT
JSON_OBJECT(
'name', U.`name`,
'order', O.`id`
) AS `result`
FROM `jsql_catalog`.`f_users` AS U,
`jsql_catalog`.`f_orders` AS O
WHERE O.`user_id` = U.`id`
ORDER BY O.`id` ASC
LIMIT 1000

-- ========== 4. Consultas · Model.Query con from sin esquema [pass]

-- QUERY
SELECT
JSON_OBJECT(
'name', R.`name`
) AS `result`
FROM `jsql_catalog`.`f_roles` AS R
ORDER BY R.`name` ASC
LIMIT 1000

-- ========== 4. Consultas · Model.Query con from + join + groups [pass]

-- QUERY
SELECT
JSON_OBJECT(
'name', ANY_VALUE(U.`name`),
'n', COUNT(O.`id`)
) AS `result`
FROM `jsql_catalog`.`f_users` AS U
INNER JOIN `jsql_catalog`.`f_orders` AS O
  ON O.`user_id` = U.`id`
GROUP BY U.`name`
ORDER BY U.`name` ASC
LIMIT 1000

-- ========== 4. Consultas · DB.Query consulta (select, from, join, group by, order by) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'name', ANY_VALUE(U.`name`),
'n', COUNT(O.`id`)
) AS `result`
FROM `jsql_catalog`.`f_users` AS U
LEFT JOIN `jsql_catalog`.`f_orders` AS O
  ON O.`user_id` = U.`id`
GROUP BY U.`name`
ORDER BY U.`name` ASC
LIMIT 1000

-- ========== 4. Consultas · DB.Query consulta (where + and/or de primer nivel, limit, offset) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id', A.`id`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) > 18
  OR A.`name` = 'Luis'
ORDER BY A.`id` ASC
LIMIT 2 OFFSET 1

-- ========== 4. Consultas · Model.Query con claves select / group by / order by [pass]

-- QUERY
SELECT
JSON_OBJECT(
'user_id', ANY_VALUE(A.`user_id`),
'n', COUNT(A.`id`)
) AS `result`
FROM `jsql_catalog`.`f_orders` AS A
GROUP BY A.`user_id`
HAVING COUNT(A.`id`) > 1
ORDER BY A.`user_id` ASC
LIMIT 1000

-- ========== 4. Consultas · Test (genera el SQL sin ejecutarlo) / Debug [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` = 'u1'
LIMIT 1000

-- ========== 5. Condiciones · Eq [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`name` = 'Ana'
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Neg [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`name` != 'Ana'
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Less [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) < 30
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · LessEq [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) <= 30
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · More [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) > 30
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · MoreEq [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) >= 30
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Like [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE LOWER(A.`name`) LIKE LOWER('%an%')
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · In [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` IN ('u1', 'u3')
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · NotIn [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`id` NOT IN ('u1', 'u3')
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Is (NULL) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."nickname"')) IS NULL
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · IsNot (NULL) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) IS NOT NULL
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Null [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."nickname"')) IS NULL
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · NotNull [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`email` IS NOT NULL
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Between [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) BETWEEN 18 AND 40
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · NotBetween [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) NOT BETWEEN 18 AND 40
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Where (genérico) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`name` = 'Luis'
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · Is / IsNot con valor (comparación NULL-safe) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`name` <=> 'Ana'
ORDER BY A.`id` ASC
LIMIT 1000

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE NOT (A.`name` <=> 'Ana')
ORDER BY A.`id` ASC
LIMIT 1000

-- ========== 5. Condiciones · And / Or (conectores) [pass]

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE A.`name` = 'Ana'
  OR A.`name` = 'Luis'
ORDER BY A.`id` ASC
LIMIT 1000

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"', '$."password"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."email"', A.`email`
) AS `result`
FROM `jsql_catalog`.`f_users` AS A
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(A.`_source`, '$."age"')) AS SIGNED) > 18
  AND LOWER(A.`name`) LIKE LOWER('%neil%')
LIMIT 1000

-- ========== 6. Comandos · Insert (RETURNING) [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('equipos', 'p4', 'Router Wi-Fi 6', '{"price":350000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p4';

-- ========== 6. Comandos · Bulk [pass]

-- BULK
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('tv', 'p5', 'Cable HDMI', '{"price":15000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p5';

-- BULK
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('tv', 'p6', 'Control', '{"price":10000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p6';

-- ========== 6. Comandos · Update + Where [pass]

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p1'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'internet',
    `name` = 'Plan 200',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."price"', CAST('99000' AS JSON),
'$."promo"', CAST('true' AS JSON)
)
  WHERE `id` = 'p1';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p1';

-- ========== 6. Comandos · Update + Where + Or (varias filas) [pass]

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p5'
  OR A.`id` = 'p6'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'tv',
    `name` = 'Cable HDMI',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."price"', CAST('15000' AS JSON),
'$."stock"', CAST('5' AS JSON)
)
  WHERE `id` = 'p5';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p5';

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'tv',
    `name` = 'Control',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."price"', CAST('10000' AS JSON),
'$."stock"', CAST('5' AS JSON)
)
  WHERE `id` = 'p6';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p6';

-- ========== 6. Comandos · Upsert (inserta y luego actualiza) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p7') THEN 'true' ELSE 'false' END AS `exists`

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`)
  VALUES ('tv', 'p7', 'Antena');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p7';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p7') THEN 'true' ELSE 'false' END AS `exists`

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p7'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'tv',
    `name` = 'Antena HD'
  WHERE `id` = 'p7';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p7';

-- ========== 6. Comandos · Return (campos como en Select: columna, atributo, alias, ruta) [pass]

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p4'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'equipos',
    `name` = 'Router Wi-Fi 6',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."price"', CAST('350000' AS JSON),
'$."specs"', CAST('{"wifi":"6"}' AS JSON),
'$."stock"', CAST('7' AS JSON)
)
  WHERE `id` = 'p4';
SELECT JSON_OBJECT('id', `id`, 'nombre', `name`, 'stock', JSON_EXTRACT(`_source`, '$."stock"'), 'wifi', JSON_EXTRACT(`_source`, '$."specs"."wifi"')) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p4';

-- ========== 6. Comandos · Delete (devuelve la fila borrada) [pass]

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p6'
LIMIT 1000

-- DELETE
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p6';
DELETE FROM `jsql_catalog`.`f_products` WHERE `id` = 'p6';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p6') THEN 'true' ELSE 'false' END AS `exists`

-- ========== 6. Comandos · DB.Query insert + bulk (con triggers JS) [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`, `_source`)
  VALUES ('equipos', 'j1', 'Mesh', '{"origin":"json","price":450000}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j1';

-- BULK
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`)
  VALUES ('equipos', 'j2', 'Extensor');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j2';

-- BULK
INSERT INTO `jsql_catalog`.`f_products`
  (`category`, `id`, `name`)
  VALUES ('equipos', 'j3', 'Splitter');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j3';

-- ========== 6. Comandos · DB.Query update + delete [pass]

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`category` = 'equipos'
  AND A.`id` IN ('j2', 'j3')
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'equipos',
    `name` = 'Extensor',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."price"', CAST('99' AS JSON),
'$."touched"', CAST('true' AS JSON)
)
  WHERE `id` = 'j2';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j2';

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = 'equipos',
    `name` = 'Splitter',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."price"', CAST('99' AS JSON),
'$."touched"', CAST('true' AS JSON)
)
  WHERE `id` = 'j3';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j3';

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'j3'
LIMIT 1000

-- DELETE
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j3';
DELETE FROM `jsql_catalog`.`f_products` WHERE `id` = 'j3';

-- ========== 6. Comandos · DB.Query upsert (inserta, actualiza y exige where) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'j4') THEN 'true' ELSE 'false' END AS `exists`

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`id`, `name`, `_source`)
  VALUES ('j4', 'Repetidor', '{"both":true,"path":"insert"}');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j4';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'j4') THEN 'true' ELSE 'false' END AS `exists`

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'j4'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = NULL,
    `name` = 'Repetidor Pro',
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."both"', CAST('true' AS JSON),
'$."path"', CAST('"update"' AS JSON)
)
  WHERE `id` = 'j4';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'j4';

-- ========== 6. Comandos · Update / Delete con limit (por defecto, n y 0 = todas) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_many` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `name` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_many_status_idx` ON `jsql_catalog`.`f_many` (`status`);
CREATE INDEX `f_many__idx_idx` ON `jsql_catalog`.`f_many` (`_idx`);

-- INSERT
INSERT INTO `jsql_catalog`.`f_many`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368689', 'm1', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm1';

-- INSERT
INSERT INTO `jsql_catalog`.`f_many`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368690', 'm2', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm2';

-- INSERT
INSERT INTO `jsql_catalog`.`f_many`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368691', 'm3', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm3';

-- INSERT
INSERT INTO `jsql_catalog`.`f_many`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368692', 'm4', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm4';

-- INSERT
INSERT INTO `jsql_catalog`.`f_many`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368694', 'm5', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm5';

-- INSERT
INSERT INTO `jsql_catalog`.`f_many`
  (`_idx`, `id`, `name`)
  VALUES ('1790553368695', 'm6', 'x');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm6';

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_many` AS A
WHERE A.`name` = 'x'
LIMIT 2

-- UPDATE
UPDATE `jsql_catalog`.`f_many`
  SET `created_at` = NULL,
    `name` = 'y',
    `status` = 'active',
    `updated_at` = NULL
  WHERE `id` = 'm1';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm1';

-- UPDATE
UPDATE `jsql_catalog`.`f_many`
  SET `created_at` = NULL,
    `name` = 'y',
    `status` = 'active',
    `updated_at` = NULL
  WHERE `id` = 'm2';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm2';

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_many` AS A
WHERE A.`name` = 'x'
LIMIT 3

-- UPDATE
UPDATE `jsql_catalog`.`f_many`
  SET `created_at` = NULL,
    `name` = 'y',
    `status` = 'active',
    `updated_at` = NULL
  WHERE `id` = 'm3';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm3';

-- UPDATE
UPDATE `jsql_catalog`.`f_many`
  SET `created_at` = NULL,
    `name` = 'y',
    `status` = 'active',
    `updated_at` = NULL
  WHERE `id` = 'm4';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm4';

-- UPDATE
UPDATE `jsql_catalog`.`f_many`
  SET `created_at` = NULL,
    `name` = 'y',
    `status` = 'active',
    `updated_at` = NULL
  WHERE `id` = 'm5';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm5';

-- QUERY
SELECT COUNT(*) AS `count`
FROM `jsql_catalog`.`f_many` AS A
WHERE A.`name` = 'x'

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_many` AS A
WHERE A.`name` IN ('x', 'y')

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm1';
DELETE FROM `jsql_catalog`.`f_many` WHERE `id` = 'm1';

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm2';
DELETE FROM `jsql_catalog`.`f_many` WHERE `id` = 'm2';

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm3';
DELETE FROM `jsql_catalog`.`f_many` WHERE `id` = 'm3';

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm4';
DELETE FROM `jsql_catalog`.`f_many` WHERE `id` = 'm4';

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm5';
DELETE FROM `jsql_catalog`.`f_many` WHERE `id` = 'm5';

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_many` WHERE `id` = 'm6';
DELETE FROM `jsql_catalog`.`f_many` WHERE `id` = 'm6';

-- QUERY
SELECT COUNT(*) AS `count`
FROM `jsql_catalog`.`f_many` AS A

-- ========== 6. Comandos · Upsert fluido sin where (devuelve error) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p9') THEN 'true' ELSE 'false' END AS `exists`

-- ========== 6. Comandos · Test (no ejecuta) + ToJson [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`id`, `name`)
  VALUES ('p8', 'No se guarda');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 'p8';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 'p8') THEN 'true' ELSE 'false' END AS `exists`

-- ========== 7. Triggers · Before/After Insert, Update, Delete e InsertOrUpdate [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`f_events` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `status` VARCHAR(255) DEFAULT 'active',
  `id` VARCHAR(80) NOT NULL,
  `_source` JSON DEFAULT (JSON_OBJECT()),
  `name` VARCHAR(80) DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`id`)
);
CREATE INDEX `f_events_status_idx` ON `jsql_catalog`.`f_events` (`status`);
CREATE INDEX `f_events__idx_idx` ON `jsql_catalog`.`f_events` (`_idx`);

-- INSERT
INSERT INTO `jsql_catalog`.`f_events`
  (`_idx`, `id`, `name`, `_source`)
  VALUES ('1790553368726', 'e1', 'alta', '{"stage":"before_insert","touched":true}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_events` WHERE `id` = 'e1';

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_events` AS A
WHERE A.`id` = 'e1'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_events`
  SET `created_at` = NULL,
    `name` = 'cambio',
    `status` = 'active',
    `updated_at` = NULL,
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."stage"', CAST('"before_update"' AS JSON),
'$."touched"', CAST('true' AS JSON)
)
  WHERE `id` = 'e1';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_events` WHERE `id` = 'e1';

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_events` AS A
WHERE A.`id` = 'e1'
LIMIT 1000

-- DELETE
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_events` WHERE `id` = 'e1';
DELETE FROM `jsql_catalog`.`f_events` WHERE `id` = 'e1';

-- ========== 7. Triggers · Trigger del comando + error que aborta [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_events`
  (`_idx`, `id`, `name`, `_source`)
  VALUES ('1790553368730', 'e2', 'cmd', '{"source":"command","stage":"before_insert","touched":true}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_events` WHERE `id` = 'e2';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_events` AS A
WHERE A.`id` = 'e3') THEN 'true' ELSE 'false' END AS `exists`

-- ========== 7. Triggers · Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_events`
  (`_idx`, `id`, `name`, `_source`)
  VALUES ('1790553368732', 'e4', 'js', '{"js":"before_insert","stage":"before_insert","touched":true}');
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_events` WHERE `id` = 'e4';

-- QUERY
SELECT
JSON_SET(JSON_REMOVE(COALESCE(A.`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', A.`created_at`,
'$."updated_at"', A.`updated_at`,
'$."status"', A.`status`,
'$."id"', A.`id`,
'$."name"', A.`name`
) AS `result`
FROM `jsql_catalog`.`f_events` AS A
WHERE A.`id` = 'e4'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_events`
  SET `created_at` = NULL,
    `name` = 'js2',
    `status` = 'active',
    `updated_at` = NULL,
    `_source` = JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."js"', CAST('"before_update"' AS JSON),
'$."stage"', CAST('"before_update"' AS JSON),
'$."touched"', CAST('true' AS JSON)
)
  WHERE `id` = 'e4';
SELECT JSON_SET(JSON_REMOVE(COALESCE(`_source`, JSON_OBJECT()), '$."_idx"'),
'$."created_at"', `created_at`,
'$."updated_at"', `updated_at`,
'$."status"', `status`,
'$."id"', `id`,
'$."name"', `name`
) AS `result` FROM `jsql_catalog`.`f_events` WHERE `id` = 'e4';

-- ========== 8. Transacciones · NewTx + ExecTx + Rollback [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`id`, `name`)
  VALUES ('t1', 'rollback');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 't1';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 't1') THEN 'true' ELSE 'false' END AS `exists`

-- ========== 8. Transacciones · NewTx + ExecTx + Commit [pass]

-- INSERT
INSERT INTO `jsql_catalog`.`f_products`
  (`id`, `name`)
  VALUES ('t2', 'commit');
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 't2';

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 't2'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`f_products`
  SET `category` = NULL,
    `name` = 'commit 2'
  WHERE `id` = 't2';
SELECT JSON_SET(COALESCE(`_source`, JSON_OBJECT()),
'$."id"', `id`,
'$."name"', `name`,
'$."category"', `category`
) AS `result` FROM `jsql_catalog`.`f_products` WHERE `id` = 't2';

-- QUERY
SELECT
JSON_SET(COALESCE(A.`_source`, JSON_OBJECT()),
'$."id"', A.`id`,
'$."name"', A.`name`,
'$."category"', A.`category`
) AS `result`
FROM `jsql_catalog`.`f_products` AS A
WHERE A.`id` = 't2'
LIMIT 1

-- ========== 9. Series · DefineSeries [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS `jsql_catalog`;
CREATE TABLE IF NOT EXISTS `jsql_catalog`.`series` (
  `created_at` DATETIME(6) DEFAULT NULL,
  `updated_at` DATETIME(6) DEFAULT NULL,
  `tag` VARCHAR(80) NOT NULL,
  `format` VARCHAR(255) DEFAULT NULL,
  `value` BIGINT DEFAULT NULL,
  `_idx` VARCHAR(80) DEFAULT NULL,
  PRIMARY KEY (`tag`)
);
CREATE INDEX `series__idx_idx` ON `jsql_catalog`.`series` (`_idx`);

-- ========== 9. Series · SetSeries + GetSeries [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice') THEN 'true' ELSE 'false' END AS `exists`

-- INSERT
INSERT INTO `jsql_catalog`.`series`
  (`_idx`, `created_at`, `format`, `tag`, `updated_at`, `value`)
  VALUES ('1790553368752', '2026-09-27 18:56:08.752898', 'FAC-%05d', 'invoice', '2026-09-27 18:56:08.752898', 10);
SELECT JSON_OBJECT('created_at', `created_at`, 'updated_at', `updated_at`, 'tag', `tag`, 'format', `format`, 'value', `value`) AS `result` FROM `jsql_catalog`.`series` WHERE `tag` = 'invoice';

-- QUERY
SELECT
JSON_OBJECT('created_at', A.`created_at`, 'updated_at', A.`updated_at`, 'tag', A.`tag`, 'format', A.`format`, 'value', A.`value`) AS `result`
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice'
LIMIT 1

-- ========== 9. Series · GenValue + GenSerie [pass]

-- QUERY
SELECT
JSON_OBJECT('created_at', A.`created_at`, 'updated_at', A.`updated_at`, 'tag', A.`tag`, 'format', A.`format`, 'value', A.`value`) AS `result`
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`series`
  SET `created_at` = '2026-09-27 18:56:08.752898',
    `format` = 'FAC-%05d',
    `updated_at` = '2026-09-27 18:56:08.755089',
    `value` = 11
  WHERE `tag` = 'invoice';
SELECT JSON_OBJECT('created_at', `created_at`, 'updated_at', `updated_at`, 'tag', `tag`, 'format', `format`, 'value', `value`) AS `result` FROM `jsql_catalog`.`series` WHERE `tag` = 'invoice';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice') THEN 'true' ELSE 'false' END AS `exists`

-- QUERY
SELECT
JSON_OBJECT('created_at', A.`created_at`, 'updated_at', A.`updated_at`, 'tag', A.`tag`, 'format', A.`format`, 'value', A.`value`) AS `result`
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice'
LIMIT 1000

-- UPDATE
UPDATE `jsql_catalog`.`series`
  SET `created_at` = '2026-09-27 18:56:08.752898',
    `format` = 'FAC-%05d',
    `updated_at` = '2026-09-27 18:56:08.756990',
    `value` = 12
  WHERE `tag` = 'invoice';
SELECT JSON_OBJECT('created_at', `created_at`, 'updated_at', `updated_at`, 'tag', `tag`, 'format', `format`, 'value', `value`) AS `result` FROM `jsql_catalog`.`series` WHERE `tag` = 'invoice';

-- ========== 9. Series · DeleteSeries [pass]

-- QUERY
SELECT
JSON_OBJECT('created_at', A.`created_at`, 'updated_at', A.`updated_at`, 'tag', A.`tag`, 'format', A.`format`, 'value', A.`value`) AS `result`
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice'
LIMIT 1000

-- DELETE
SELECT JSON_OBJECT('created_at', `created_at`, 'updated_at', `updated_at`, 'tag', `tag`, 'format', `format`, 'value', `value`) AS `result` FROM `jsql_catalog`.`series` WHERE `tag` = 'invoice';
DELETE FROM `jsql_catalog`.`series` WHERE `tag` = 'invoice';

-- QUERY
SELECT
JSON_OBJECT('created_at', A.`created_at`, 'updated_at', A.`updated_at`, 'tag', A.`tag`, 'format', A.`format`, 'value', A.`value`) AS `result`
FROM `jsql_catalog`.`series` AS A
WHERE A.`tag` = 'invoice'
LIMIT 1


-- ========== 2. Definición de modelos (DDL) · Define (declarativo) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_products (
  id VARCHAR(80) DEFAULT NULL,
  name VARCHAR(255) DEFAULT NULL,
  category VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}'
);

ALTER TABLE jsql_catalog.f_products ADD CONSTRAINT jsql_catalog_f_products_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_products_category_idx ON jsql_catalog.f_products USING BTREE (category);

-- ========== 2. Definición de modelos (DDL) · DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_roles (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_roles ADD CONSTRAINT jsql_catalog_f_roles_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_roles_status_idx ON jsql_catalog.f_roles USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_roles__idx_idx ON jsql_catalog.f_roles USING BTREE (_idx);

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_users (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  email VARCHAR(255) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_users ADD CONSTRAINT jsql_catalog_f_users_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX IF NOT EXISTS jsql_catalog_f_users_email_key ON jsql_catalog.f_users (email);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_users_status_idx ON jsql_catalog.f_users USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_users__idx_idx ON jsql_catalog.f_users USING BTREE (_idx);

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_users_f_roles (
  user_id VARCHAR(80) DEFAULT NULL,
  role_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_users_f_roles ADD CONSTRAINT jsql_catalog_f_users_f_roles_pkey PRIMARY KEY (user_id, role_id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_users_f_roles__idx_idx ON jsql_catalog.f_users_f_roles USING BTREE (_idx);
ALTER TABLE jsql_catalog.f_users_f_roles ADD CONSTRAINT fk_jsql_catalog_f_users_f_roles_jsql_catalog_f_users
  FOREIGN KEY (user_id)
  REFERENCES jsql_catalog.f_users (id)
  ON DELETE CASCADE;
ALTER TABLE jsql_catalog.f_users_f_roles ADD CONSTRAINT fk_jsql_catalog_f_users_f_roles_jsql_catalog_f_roles
  FOREIGN KEY (role_id)
  REFERENCES jsql_catalog.f_roles (id)
  ON DELETE CASCADE;

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_doc_types (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  title VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_doc_types ADD CONSTRAINT jsql_catalog_f_doc_types_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_doc_types_status_idx ON jsql_catalog.f_doc_types USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_doc_types__idx_idx ON jsql_catalog.f_doc_types USING BTREE (_idx);

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_orders (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  user_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_orders ADD CONSTRAINT jsql_catalog_f_orders_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_orders_status_idx ON jsql_catalog.f_orders USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_orders__idx_idx ON jsql_catalog.f_orders USING BTREE (_idx);

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_orders_items (
  order_id VARCHAR(80) DEFAULT NULL,
  id VARCHAR(80) DEFAULT NULL,
  product VARCHAR(255) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_orders_items ADD CONSTRAINT jsql_catalog_f_orders_items_pkey PRIMARY KEY (id);
ALTER TABLE jsql_catalog.f_orders_items ADD CONSTRAINT fk_jsql_catalog_f_orders_items_jsql_catalog_f_orders
  FOREIGN KEY (order_id)
  REFERENCES jsql_catalog.f_orders (id)
  ON DELETE CASCADE;

-- ========== 2. Definición de modelos (DDL) · DefineTenantModel / DefineProjectModel [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_tenant (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  tenant_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_tenant ADD CONSTRAINT jsql_catalog_f_tenant_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_tenant_status_idx ON jsql_catalog.f_tenant USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_tenant__idx_idx ON jsql_catalog.f_tenant USING BTREE (_idx);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_tenant_tenant_id_idx ON jsql_catalog.f_tenant USING BTREE (tenant_id);

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_project (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  project_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_project ADD CONSTRAINT jsql_catalog_f_project_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_project_status_idx ON jsql_catalog.f_project USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_project__idx_idx ON jsql_catalog.f_project USING BTREE (_idx);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_project_project_id_idx ON jsql_catalog.f_project USING BTREE (project_id);

-- ========== 2. Definición de modelos (DDL) · DefineRequired (rechaza el insert sin el campo) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_required (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(255) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_required ADD CONSTRAINT jsql_catalog_f_required_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_required_status_idx ON jsql_catalog.f_required USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_required__idx_idx ON jsql_catalog.f_required USING BTREE (_idx);

-- ========== 2. Definición de modelos (DDL) · DefineForeignKeys (rechaza un hijo sin padre) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_parent (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_parent ADD CONSTRAINT jsql_catalog_f_parent_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_parent_status_idx ON jsql_catalog.f_parent USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_parent__idx_idx ON jsql_catalog.f_parent USING BTREE (_idx);

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_child (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  parent_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_child ADD CONSTRAINT jsql_catalog_f_child_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_child_status_idx ON jsql_catalog.f_child USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_child__idx_idx ON jsql_catalog.f_child USING BTREE (_idx);
ALTER TABLE jsql_catalog.f_child ADD CONSTRAINT fk_jsql_catalog_f_child_jsql_catalog_f_parent
  FOREIGN KEY (parent_id)
  REFERENCES jsql_catalog.f_parent (id)
  ON DELETE CASCADE;

-- INSERT
INSERT INTO jsql_catalog.f_parent
  (_idx, id)
VALUES
  ('1790455005747', 'p1')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_child
  (_idx, id, parent_id)
VALUES
  ('1790455005753', 'c1', 'p1')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'parent_id', parent_id
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_child
  (_idx, id, parent_id)
VALUES
  ('1790455005758', 'c2', 'missing')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'parent_id', parent_id
) AS result;

-- ========== 2. Definición de modelos (DDL) · Stricted (ignora campos desconocidos) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_strict (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_strict ADD CONSTRAINT jsql_catalog_f_strict_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_strict_status_idx ON jsql_catalog.f_strict USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_strict__idx_idx ON jsql_catalog.f_strict USING BTREE (_idx);

-- INSERT
INSERT INTO jsql_catalog.f_strict
  (_idx, id, name)
VALUES
  ('1790455005768', 's1', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_strict AS A
WHERE A.id = 's1'
LIMIT 1;

-- ========== 2. Definición de modelos (DDL) · DB.Query define (Define en JSON) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_json_model (
  id VARCHAR(80) DEFAULT NULL,
  title VARCHAR(255) DEFAULT NULL,
  _source JSONB DEFAULT '{}'
);

ALTER TABLE jsql_catalog.f_json_model ADD CONSTRAINT jsql_catalog_f_json_model_pkey PRIMARY KEY (id);

-- INSERT
INSERT INTO jsql_catalog.f_json_model
  (id, title, _source)
VALUES
  ('j1', 'desde define', '{"extra":1}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'title', title
) AS result;

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'title', A.title
) AS result
FROM jsql_catalog.f_json_model AS A
WHERE A.id = 'j1'
LIMIT 1;

-- ========== 2. Definición de modelos (DDL) · Insert (datos base) [pass]

-- INSERT
INSERT INTO jsql_catalog.f_doc_types
  (_idx, id, title)
VALUES
  ('1790455005798', 'CC', 'Cédula de ciudadanía')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'title', title
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_doc_types
  (_idx, id, title)
VALUES
  ('1790455005842', 'NIT', 'Número de identificación tributaria')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'title', title
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.email = 'ana@example.com');

-- INSERT
INSERT INTO jsql_catalog.f_users
  (_idx, email, id, name, _source)
VALUES
  ('1790455005857', 'ana@example.com', 'u1', 'Ana', '{"age":30,"password":"secret","tp_doc":"CC"}'::jsonb)
RETURNING (_source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name,
'email', email
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.email = 'luis@example.com');

-- INSERT
INSERT INTO jsql_catalog.f_users
  (_idx, email, id, name, _source)
VALUES
  ('1790455005864', 'luis@example.com', 'u2', 'Luis', '{"age":17}'::jsonb)
RETURNING (_source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name,
'email', email
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.email = 'marta@example.com');

-- INSERT
INSERT INTO jsql_catalog.f_users
  (_idx, email, id, name, _source)
VALUES
  ('1790455005865', 'marta@example.com', 'u3', 'Marta O''Neil', '{"age":45,"tp_doc":"NIT"}'::jsonb)
RETURNING (_source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name,
'email', email
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_roles
  (_idx, id, name)
VALUES
  ('1790455005865', 'r1', 'admin')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_roles
  (_idx, id, name)
VALUES
  ('1790455005870', 'r2', 'editor')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_orders
  (_idx, id, user_id, _source)
VALUES
  ('1790455005877', 'o1', 'u1', '{"amount":100.5}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'user_id', user_id
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_orders
  (_idx, id, user_id, _source)
VALUES
  ('1790455005881', 'o2', 'u1', '{"amount":200}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'user_id', user_id
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_orders
  (_idx, id, user_id, _source)
VALUES
  ('1790455005882', 'o3', 'u3', '{"amount":50}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'user_id', user_id
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_orders_items
  (id, order_id, product)
VALUES
  ('i1', 'o1', 'router')
RETURNING id, product;

-- INSERT
INSERT INTO jsql_catalog.f_orders_items
  (id, order_id, product)
VALUES
  ('i2', 'o1', 'cable')
RETURNING id, product;

-- INSERT
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('internet', 'p1', 'Plan 200', '{"price":90000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('internet', 'p2', 'Plan 500', '{"price":150000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('tv', 'p3', 'Decoder', '{"price":20000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_users_f_roles
  (_idx, role_id, user_id)
VALUES
  ('1790455005893', 'r1', 'u1')
RETURNING user_id, role_id;

-- INSERT
INSERT INTO jsql_catalog.f_users_f_roles
  (_idx, role_id, user_id)
VALUES
  ('1790455005894', 'r2', 'u1')
RETURNING user_id, role_id;

-- INSERT
INSERT INTO jsql_catalog.f_users_f_roles
  (_idx, role_id, user_id)
VALUES
  ('1790455005895', 'r2', 'u2')
RETURNING user_id, role_id;

-- ========== 2. Definición de modelos (DDL) · DefineUnique (rechaza duplicados en insert, bulk y update) [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.email = 'ana@example.com');

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.email = 'same@example.com');

-- BULK
INSERT INTO jsql_catalog.f_users
  (_idx, email, id, name)
VALUES
  ('1790455005897', 'same@example.com', 'u8', 'Uno')
RETURNING (_source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name,
'email', email
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.email = 'same@example.com');

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u2'
LIMIT 1000;

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.email = 'ana@example.com'
LIMIT 2;

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u2'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_users
SET
  created_at = NULL,
  email = 'luis@example.com',
  name = 'Luis',
  status = 'active',
  updated_at = NULL,
  _source = COALESCE(_source, '{}'::jsonb) || '{"age":17}'::jsonb
WHERE id = 'u2'
RETURNING (_source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name,
'email', email
) AS result;

-- QUERY
SELECT COUNT(*)
FROM jsql_catalog.f_users AS A;

-- ========== 3. Relaciones y campos calculados · Detail en el select (DefineDetail) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id
) AS result
FROM jsql_catalog.f_orders AS A
WHERE A.id = 'o1'
LIMIT 1;

-- QUERY
SELECT
A.product
FROM jsql_catalog.f_orders_items AS A
WHERE A.order_id = 'o1'
LIMIT 30;

-- ========== 3. Relaciones y campos calculados · Model.Detail + Query.Detail [pass]

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'user_id', A.user_id
) AS result
FROM jsql_catalog.f_orders AS A
WHERE A.id = 'o1'
LIMIT 1;

-- QUERY
SELECT
A.product
FROM jsql_catalog.f_orders_items AS A
WHERE A.order_id = 'o1'
LIMIT 30;

-- ========== 3. Relaciones y campos calculados · Master en el select (DefineMaster) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u1'
LIMIT 1;

-- QUERY
SELECT
jsonb_build_object(
'name', A.name
) AS result
FROM jsql_catalog.f_roles AS A
INNER JOIN jsql_catalog.f_users_f_roles AS B
  ON B.role_id = A.id
WHERE B.user_id = 'u1'
LIMIT 1000;

-- ========== 3. Relaciones y campos calculados · Master 1 a 1 (rows = 1) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u2'
LIMIT 1;

-- QUERY
SELECT
jsonb_build_object(
'name', A.name
) AS result
FROM jsql_catalog.f_roles AS A
INNER JOIN jsql_catalog.f_users_f_roles AS B
  ON B.role_id = A.id
WHERE B.user_id = 'u2'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · Model.Master + Model.Bridge [pass]

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_roles AS A
INNER JOIN jsql_catalog.f_users_f_roles AS B
  ON A.id = B.role_id
WHERE B.user_id = 'u2'
LIMIT 1000;

-- ========== 3. Relaciones y campos calculados · DefineRollup row (tp_doc → title) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'tp_doc', A._source->'tp_doc'
) AS result
FROM jsql_catalog.f_users AS A
ORDER BY A.id ASC
LIMIT 1000;

-- QUERY
SELECT
jsonb_build_object(
'title', A.title
) AS result
FROM jsql_catalog.f_doc_types AS A
WHERE A.id = 'CC'
LIMIT 1;

-- QUERY
SELECT
jsonb_build_object(
'title', A.title
) AS result
FROM jsql_catalog.f_doc_types AS A
WHERE A.id = NULL
LIMIT 1;

-- QUERY
SELECT
jsonb_build_object(
'title', A.title
) AS result
FROM jsql_catalog.f_doc_types AS A
WHERE A.id = 'NIT'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · DefineRollup (count, sum, object) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u3'
LIMIT 1;

-- QUERY
SELECT COUNT(*)
FROM jsql_catalog.f_orders AS A
WHERE A.user_id = 'u3';

-- QUERY
SELECT
jsonb_build_object(
'amount', COALESCE(SUM((A._source->>'amount')::double precision), 0)::double precision
) AS result
FROM jsql_catalog.f_orders AS A
WHERE A.user_id = 'u3'
LIMIT 1000;

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'amount', (A._source->>'amount')::double precision
) AS result
FROM jsql_catalog.f_orders AS A
WHERE A.user_id = 'u3'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · DefineCalcFunc + Model.Calc [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u1'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · Query.Calc con script JS (DefineCalc) [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u3'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · DefineCalc (script JS) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u3'
LIMIT 1;

-- ========== 4. Consultas · Select + Where + OrderBy + All [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'name', A.name,
'age', (A._source->>'age')::bigint
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint > 18
ORDER BY (A._source->>'age')::bigint DESC
LIMIT 1000;

-- ========== 4. Consultas · One / First / Count / Exists [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u2'
LIMIT 1;

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
ORDER BY A.id ASC
LIMIT 2;

-- QUERY
SELECT COUNT(*)
FROM jsql_catalog.f_users AS A;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u9');

-- ========== 4. Consultas · Limit (paginación) / Page [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
ORDER BY A.id ASC
LIMIT 1
OFFSET 1;

-- ========== 4. Consultas · Hidden (campo oculto en la consulta y en el modelo) [pass]

-- QUERY
SELECT
(A._source - '{"email","_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u1'
LIMIT 1;

-- ========== 4. Consultas · Join (fluido) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'user', U.name
) AS result
FROM jsql_catalog.f_orders AS A
INNER JOIN jsql_catalog.f_users AS U
  ON A.user_id = U.id
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 4. Consultas · LeftJoin + GroupBy [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'n', COUNT(O.id)::bigint
) AS result
FROM jsql_catalog.f_users AS A
LEFT JOIN jsql_catalog.f_orders AS O
  ON O.user_id = A.id
GROUP BY A.id
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 4. Consultas · RightJoin / FullJoin [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', U.id,
'n', COUNT(A.id)::bigint
) AS result
FROM jsql_catalog.f_orders AS A
RIGHT JOIN jsql_catalog.f_users AS U
  ON A.user_id = U.id
GROUP BY U.id
ORDER BY U.id ASC
LIMIT 1000;

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'n', COUNT(O.id)::bigint
) AS result
FROM jsql_catalog.f_users AS A
FULL JOIN jsql_catalog.f_orders AS O
  ON O.user_id = A.id
GROUP BY A.id
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 4. Consultas · GroupBy + Having (fluido) [pass]

-- QUERY
SELECT
jsonb_build_object(
'user_id', A.user_id,
'n', COUNT(A.id)::bigint,
'total', COALESCE(SUM((A._source->>'amount')::double precision), 0)::double precision
) AS result
FROM jsql_catalog.f_orders AS A
GROUP BY A.user_id
HAVING COUNT(A.id)::bigint > 1
LIMIT 1000;

-- ========== 4. Consultas · Model.Query (JSON) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint >= 18
ORDER BY A.name DESC
LIMIT 10;

-- ========== 4. Consultas · Model.Query con from (otro modelo y alias) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', U.id,
'name', U.name
) AS result
FROM jsql_catalog.f_users AS U
WHERE (U._source->>'age')::bigint > 18
ORDER BY U.id ASC
LIMIT 1000;

-- ========== 4. Consultas · Model.Query con from sin esquema [pass]

-- QUERY
SELECT
jsonb_build_object(
'name', R.name
) AS result
FROM jsql_catalog.f_roles AS R
ORDER BY R.name ASC
LIMIT 1000;

-- ========== 4. Consultas · Model.Query con from + join + groups [pass]

-- QUERY
SELECT
jsonb_build_object(
'name', U.name,
'n', COUNT(O.id)::bigint
) AS result
FROM jsql_catalog.f_users AS U
INNER JOIN jsql_catalog.f_orders AS O
  ON O.user_id = U.id
GROUP BY U.name
ORDER BY U.name ASC
LIMIT 1000;

-- ========== 4. Consultas · DB.Query consulta (select, from, join, group by, order by) [pass]

-- QUERY
SELECT
jsonb_build_object(
'name', U.name,
'n', COUNT(O.id)::bigint
) AS result
FROM jsql_catalog.f_users AS U
LEFT JOIN jsql_catalog.f_orders AS O
  ON O.user_id = U.id
GROUP BY U.name
ORDER BY U.name ASC
LIMIT 1000;

-- ========== 4. Consultas · DB.Query consulta (where + and/or de primer nivel, limit, offset) [pass]

-- QUERY
SELECT
jsonb_build_object(
'id', A.id
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint > 18
  OR A.name = 'Luis'
ORDER BY A.id ASC
LIMIT 2
OFFSET 1;

-- ========== 4. Consultas · Model.Query con claves select / group by / order by [pass]

-- QUERY
SELECT
jsonb_build_object(
'user_id', A.user_id,
'n', COUNT(A.id)::bigint
) AS result
FROM jsql_catalog.f_orders AS A
GROUP BY A.user_id
HAVING COUNT(A.id)::bigint > 1
ORDER BY A.user_id ASC
LIMIT 1000;

-- ========== 4. Consultas · Test (genera el SQL sin ejecutarlo) / Debug [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id = 'u1'
LIMIT 1000;

-- ========== 5. Condiciones · Eq [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.name = 'Ana'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Neg [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.name != 'Ana'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Less [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint < 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · LessEq [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint <= 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · More [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint > 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · MoreEq [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint >= 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Like [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.name ILIKE '%an%'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · In [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id IN ('u1', 'u3')
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · NotIn [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.id NOT IN ('u1', 'u3')
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Is (NULL) [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A._source->>'nickname' IS NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · IsNot (NULL) [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint IS NOT NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Null [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A._source->>'nickname' IS NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · NotNull [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.email IS NOT NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Between [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint BETWEEN 18 AND 40
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · NotBetween [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint NOT BETWEEN 18 AND 40
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Where (genérico) [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.name = 'Luis'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · And / Or (conectores) [pass]

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE A.name = 'Ana'
  OR A.name = 'Luis'
ORDER BY A.id ASC
LIMIT 1000;

-- QUERY
SELECT
(A._source - '{"_idx","password"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name,
'email', A.email
) AS result
FROM jsql_catalog.f_users AS A
WHERE (A._source->>'age')::bigint > 18
  AND A.name ILIKE '%neil%'
LIMIT 1000;

-- ========== 6. Comandos · Insert (RETURNING) [pass]

-- INSERT
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('equipos', 'p4', 'Router Wi-Fi 6', '{"price":350000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · Bulk [pass]

-- BULK
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('tv', 'p5', 'Cable HDMI', '{"price":15000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- BULK
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('tv', 'p6', 'Control', '{"price":10000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · Update + Where [pass]

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p1'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'internet',
  name = 'Plan 200',
  _source = COALESCE(_source, '{}'::jsonb) || '{"price":99000,"promo":true}'::jsonb
WHERE id = 'p1'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · Update + Where + Or (varias filas) [pass]

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p5'
  OR A.id = 'p6'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'tv',
  name = 'Cable HDMI',
  _source = COALESCE(_source, '{}'::jsonb) || '{"price":15000,"stock":5}'::jsonb
WHERE id = 'p5'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'tv',
  name = 'Control',
  _source = COALESCE(_source, '{}'::jsonb) || '{"price":10000,"stock":5}'::jsonb
WHERE id = 'p6'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · Upsert (inserta y luego actualiza) [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p7');

-- INSERT
INSERT INTO jsql_catalog.f_products
  (category, id, name)
VALUES
  ('tv', 'p7', 'Antena')
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p7');

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p7'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'tv',
  name = 'Antena HD'
WHERE id = 'p7'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · Return (campos del RETURNING) [pass]

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p4'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'equipos',
  name = 'Router Wi-Fi 6',
  _source = COALESCE(_source, '{}'::jsonb) || '{"price":350000,"stock":7}'::jsonb
WHERE id = 'p4'
RETURNING id;

-- ========== 6. Comandos · Delete (devuelve la fila borrada) [pass]

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p6'
LIMIT 1000;

-- DELETE
DELETE FROM jsql_catalog.f_products
WHERE id = 'p6'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p6');

-- ========== 6. Comandos · DB.Query insert + bulk (con triggers JS) [pass]

-- INSERT
INSERT INTO jsql_catalog.f_products
  (category, id, name, _source)
VALUES
  ('equipos', 'j1', 'Mesh', '{"origin":"json","price":450000}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- BULK
INSERT INTO jsql_catalog.f_products
  (category, id, name)
VALUES
  ('equipos', 'j2', 'Extensor')
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- BULK
INSERT INTO jsql_catalog.f_products
  (category, id, name)
VALUES
  ('equipos', 'j3', 'Splitter')
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · DB.Query update + delete [pass]

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.category = 'equipos'
  AND A.id IN ('j2', 'j3')
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'equipos',
  name = 'Extensor',
  _source = COALESCE(_source, '{}'::jsonb) || '{"price":99,"touched":true}'::jsonb
WHERE id = 'j2'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = 'equipos',
  name = 'Splitter',
  _source = COALESCE(_source, '{}'::jsonb) || '{"price":99,"touched":true}'::jsonb
WHERE id = 'j3'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'j3'
LIMIT 1000;

-- DELETE
DELETE FROM jsql_catalog.f_products
WHERE id = 'j3'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · DB.Query upsert (inserta, actualiza y exige where) [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'j4');

-- INSERT
INSERT INTO jsql_catalog.f_products
  (id, name, _source)
VALUES
  ('j4', 'Repetidor', '{"both":true,"path":"insert"}'::jsonb)
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'j4');

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 'j4'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = NULL,
  name = 'Repetidor Pro',
  _source = COALESCE(_source, '{}'::jsonb) || '{"both":true,"path":"update"}'::jsonb
WHERE id = 'j4'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- ========== 6. Comandos · Update / Delete con limit (por defecto, n y 0 = todas) [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_many (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_many ADD CONSTRAINT jsql_catalog_f_many_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_many_status_idx ON jsql_catalog.f_many USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_many__idx_idx ON jsql_catalog.f_many USING BTREE (_idx);

-- INSERT
INSERT INTO jsql_catalog.f_many
  (_idx, id, name)
VALUES
  ('1790455005944', 'm1', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_many
  (_idx, id, name)
VALUES
  ('1790455005946', 'm2', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_many
  (_idx, id, name)
VALUES
  ('1790455005947', 'm3', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_many
  (_idx, id, name)
VALUES
  ('1790455005947', 'm4', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_many
  (_idx, id, name)
VALUES
  ('1790455005948', 'm5', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- INSERT
INSERT INTO jsql_catalog.f_many
  (_idx, id, name)
VALUES
  ('1790455005949', 'm6', 'x')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_many AS A
WHERE A.name = 'x'
LIMIT 2;

-- UPDATE
UPDATE jsql_catalog.f_many
SET
  created_at = NULL,
  name = 'y',
  status = 'active',
  updated_at = NULL
WHERE id = 'm1'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- UPDATE
UPDATE jsql_catalog.f_many
SET
  created_at = NULL,
  name = 'y',
  status = 'active',
  updated_at = NULL
WHERE id = 'm2'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_many AS A
WHERE A.name = 'x'
LIMIT 3;

-- UPDATE
UPDATE jsql_catalog.f_many
SET
  created_at = NULL,
  name = 'y',
  status = 'active',
  updated_at = NULL
WHERE id = 'm3'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- UPDATE
UPDATE jsql_catalog.f_many
SET
  created_at = NULL,
  name = 'y',
  status = 'active',
  updated_at = NULL
WHERE id = 'm4'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- UPDATE
UPDATE jsql_catalog.f_many
SET
  created_at = NULL,
  name = 'y',
  status = 'active',
  updated_at = NULL
WHERE id = 'm5'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT COUNT(*)
FROM jsql_catalog.f_many AS A
WHERE A.name = 'x';

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_many AS A
WHERE A.name IN ('x', 'y');

-- DELETE
DELETE FROM jsql_catalog.f_many
WHERE id = 'm6'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- DELETE
DELETE FROM jsql_catalog.f_many
WHERE id = 'm1'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- DELETE
DELETE FROM jsql_catalog.f_many
WHERE id = 'm2'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- DELETE
DELETE FROM jsql_catalog.f_many
WHERE id = 'm3'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- DELETE
DELETE FROM jsql_catalog.f_many
WHERE id = 'm4'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- DELETE
DELETE FROM jsql_catalog.f_many
WHERE id = 'm5'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT COUNT(*)
FROM jsql_catalog.f_many AS A;

-- ========== 6. Comandos · Upsert fluido sin where (devuelve error) [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p9');

-- ========== 6. Comandos · Test (no ejecuta) + ToJson [pass]

-- INSERT
INSERT INTO jsql_catalog.f_products
  (id, name)
VALUES
  ('p8', 'No se guarda')
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 'p8');

-- ========== 7. Triggers · Before/After Insert, Update, Delete e InsertOrUpdate [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.f_events (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.f_events ADD CONSTRAINT jsql_catalog_f_events_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_events_status_idx ON jsql_catalog.f_events USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_catalog_f_events__idx_idx ON jsql_catalog.f_events USING BTREE (_idx);

-- INSERT
INSERT INTO jsql_catalog.f_events
  (_idx, id, name, _source)
VALUES
  ('1790455005962', 'e1', 'alta', '{"stage":"before_insert","touched":true}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_events AS A
WHERE A.id = 'e1'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_events
SET
  created_at = NULL,
  name = 'cambio',
  status = 'active',
  updated_at = NULL,
  _source = COALESCE(_source, '{}'::jsonb) || '{"stage":"before_update","touched":true}'::jsonb
WHERE id = 'e1'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_events AS A
WHERE A.id = 'e1'
LIMIT 1000;

-- DELETE
DELETE FROM jsql_catalog.f_events
WHERE id = 'e1'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- ========== 7. Triggers · Trigger del comando + error que aborta [pass]

-- INSERT
INSERT INTO jsql_catalog.f_events
  (_idx, id, name, _source)
VALUES
  ('1790455005970', 'e2', 'cmd', '{"source":"command","stage":"before_insert","touched":true}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_events AS A
WHERE A.id = 'e3');

-- ========== 7. Triggers · Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) [pass]

-- INSERT
INSERT INTO jsql_catalog.f_events
  (_idx, id, name, _source)
VALUES
  ('1790455005971', 'e4', 'js', '{"js":"before_insert","stage":"before_insert","touched":true}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- QUERY
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'name', A.name
) AS result
FROM jsql_catalog.f_events AS A
WHERE A.id = 'e4'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_events
SET
  created_at = NULL,
  name = 'js2',
  status = 'active',
  updated_at = NULL,
  _source = COALESCE(_source, '{}'::jsonb) || '{"js":"before_update","stage":"before_update","touched":true}'::jsonb
WHERE id = 'e4'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- ========== 8. Transacciones · NewTx + ExecTx + Rollback [pass]

-- INSERT
INSERT INTO jsql_catalog.f_products
  (id, name)
VALUES
  ('t1', 'rollback')
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.f_products AS A
WHERE A.id = 't1');

-- ========== 8. Transacciones · NewTx + ExecTx + Commit [pass]

-- INSERT
INSERT INTO jsql_catalog.f_products
  (id, name)
VALUES
  ('t2', 'commit')
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 't2'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.f_products
SET
  category = NULL,
  name = 'commit 2'
WHERE id = 't2'
RETURNING _source ||
jsonb_build_object(
'id', id,
'name', name,
'category', category
) AS result;

-- QUERY
SELECT
A._source ||
jsonb_build_object(
'id', A.id,
'name', A.name,
'category', A.category
) AS result
FROM jsql_catalog.f_products AS A
WHERE A.id = 't2'
LIMIT 1;

-- ========== 9. Series · DefineSeries [pass]

-- DDL
CREATE SCHEMA IF NOT EXISTS jsql_catalog;

CREATE TABLE IF NOT EXISTS jsql_catalog.series (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  tag VARCHAR(80) DEFAULT NULL,
  format VARCHAR(255) DEFAULT NULL,
  value BIGINT DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_catalog.series ADD CONSTRAINT jsql_catalog_series_pkey PRIMARY KEY (tag);
CREATE INDEX IF NOT EXISTS jsql_catalog_series__idx_idx ON jsql_catalog.series USING BTREE (_idx);

-- ========== 9. Series · SetSeries + GetSeries [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice');

-- INSERT
INSERT INTO jsql_catalog.series
  (_idx, created_at, format, tag, updated_at, value)
VALUES
  ('1790455005984', '2026-09-26 15:36:45', 'FAC-%05d', 'invoice', '2026-09-26 15:36:45', 10)
RETURNING created_at, updated_at, tag, format, value;

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice'
LIMIT 1;

-- ========== 9. Series · GenValue + GenSerie [pass]

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.series
SET
  created_at = '2026-09-26 15:36:45',
  format = 'FAC-%05d',
  updated_at = '2026-09-26 15:36:45',
  value = 11
WHERE tag = 'invoice'
RETURNING created_at, updated_at, tag, format, value;

-- QUERY
SELECT EXISTS(SELECT 1
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice');

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice'
LIMIT 1000;

-- UPDATE
UPDATE jsql_catalog.series
SET
  created_at = '2026-09-26 15:36:45',
  format = 'FAC-%05d',
  updated_at = '2026-09-26 15:36:45',
  value = 12
WHERE tag = 'invoice'
RETURNING created_at, updated_at, tag, format, value;

-- ========== 9. Series · DeleteSeries [pass]

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice'
LIMIT 1000;

-- DELETE
DELETE FROM jsql_catalog.series
WHERE tag = 'invoice'
RETURNING created_at, updated_at, tag, format, value;

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM jsql_catalog.series AS A
WHERE A.tag = 'invoice'
LIMIT 1;


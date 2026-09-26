-- ========== 2. Definición de modelos (DDL) · Define (declarativo) [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_products (
  id TEXT DEFAULT NULL,
  name TEXT DEFAULT NULL,
  category TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_products_category_idx ON f_products (category);

-- ========== 2. Definición de modelos (DDL) · DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_roles (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_roles_status_idx ON f_roles (status);
CREATE INDEX IF NOT EXISTS f_roles__idx_idx ON f_roles (_idx);

-- DDL
CREATE TABLE IF NOT EXISTS f_users (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  email TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS f_users_email_key ON f_users (email);
CREATE INDEX IF NOT EXISTS f_users_status_idx ON f_users (status);
CREATE INDEX IF NOT EXISTS f_users__idx_idx ON f_users (_idx);

-- DDL
CREATE TABLE IF NOT EXISTS f_users_f_roles (
  user_id TEXT DEFAULT NULL,
  role_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES f_users (id) ON DELETE CASCADE,
  FOREIGN KEY (role_id) REFERENCES f_roles (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS f_users_f_roles__idx_idx ON f_users_f_roles (_idx);

-- DDL
CREATE TABLE IF NOT EXISTS f_doc_types (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  title TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_doc_types_status_idx ON f_doc_types (status);
CREATE INDEX IF NOT EXISTS f_doc_types__idx_idx ON f_doc_types (_idx);

-- DDL
CREATE TABLE IF NOT EXISTS f_orders (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  user_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_orders_status_idx ON f_orders (status);
CREATE INDEX IF NOT EXISTS f_orders__idx_idx ON f_orders (_idx);

-- DDL
CREATE TABLE IF NOT EXISTS f_orders_items (
  order_id TEXT DEFAULT NULL,
  id TEXT DEFAULT NULL,
  product TEXT DEFAULT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (order_id) REFERENCES f_orders (id) ON DELETE CASCADE
);


-- ========== 2. Definición de modelos (DDL) · DefineTenantModel / DefineProjectModel [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_tenant (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  tenant_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_tenant_status_idx ON f_tenant (status);
CREATE INDEX IF NOT EXISTS f_tenant__idx_idx ON f_tenant (_idx);
CREATE INDEX IF NOT EXISTS f_tenant_tenant_id_idx ON f_tenant (tenant_id);

-- DDL
CREATE TABLE IF NOT EXISTS f_project (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  project_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_project_status_idx ON f_project (status);
CREATE INDEX IF NOT EXISTS f_project__idx_idx ON f_project (_idx);
CREATE INDEX IF NOT EXISTS f_project_project_id_idx ON f_project (project_id);

-- ========== 2. Definición de modelos (DDL) · DefineRequired (rechaza el insert sin el campo) [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_required (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_required_status_idx ON f_required (status);
CREATE INDEX IF NOT EXISTS f_required__idx_idx ON f_required (_idx);

-- ========== 2. Definición de modelos (DDL) · DefineForeignKeys (rechaza un hijo sin padre) [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_parent (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_parent_status_idx ON f_parent (status);
CREATE INDEX IF NOT EXISTS f_parent__idx_idx ON f_parent (_idx);

-- DDL
CREATE TABLE IF NOT EXISTS f_child (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  parent_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (parent_id) REFERENCES f_parent (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS f_child_status_idx ON f_child (status);
CREATE INDEX IF NOT EXISTS f_child__idx_idx ON f_child (_idx);

-- INSERT
INSERT INTO f_parent
  (_idx, id)
VALUES
  ('1790448435263', 'p1')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id
) AS result;

-- INSERT
INSERT INTO f_child
  (_idx, id, parent_id)
VALUES
  ('1790448435263', 'c1', 'p1')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."parent_id"', parent_id
) AS result;

-- INSERT
INSERT INTO f_child
  (_idx, id, parent_id)
VALUES
  ('1790448435263', 'c2', 'missing')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."parent_id"', parent_id
) AS result;

-- ========== 2. Definición de modelos (DDL) · Stricted (ignora campos desconocidos) [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_strict (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_strict_status_idx ON f_strict (status);
CREATE INDEX IF NOT EXISTS f_strict__idx_idx ON f_strict (_idx);

-- INSERT
INSERT INTO f_strict
  (_idx, id, name)
VALUES
  ('1790448435264', 's1', 'x')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name
) AS result
FROM f_strict AS A
WHERE A.id = 's1'
LIMIT 1;

-- ========== 2. Definición de modelos (DDL) · Insert (datos base) [pass]

-- INSERT
INSERT INTO f_doc_types
  (_idx, id, title)
VALUES
  ('1790448435265', 'CC', 'Cédula de ciudadanía')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."title"', title
) AS result;

-- INSERT
INSERT INTO f_doc_types
  (_idx, id, title)
VALUES
  ('1790448435265', 'NIT', 'Número de identificación tributaria')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."title"', title
) AS result;

-- INSERT
INSERT INTO f_users
  (_idx, email, id, name, _source)
VALUES
  ('1790448435265', 'ana@example.com', 'u1', 'Ana', '{"age":30,"password":"secret","tp_doc":"CC"}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name,
'$."email"', email
) AS result;

-- INSERT
INSERT INTO f_users
  (_idx, email, id, name, _source)
VALUES
  ('1790448435265', 'luis@example.com', 'u2', 'Luis', '{"age":17}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name,
'$."email"', email
) AS result;

-- INSERT
INSERT INTO f_users
  (_idx, email, id, name, _source)
VALUES
  ('1790448435265', 'marta@example.com', 'u3', 'Marta O''Neil', '{"age":45,"tp_doc":"NIT"}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name,
'$."email"', email
) AS result;

-- INSERT
INSERT INTO f_roles
  (_idx, id, name)
VALUES
  ('1790448435265', 'r1', 'admin')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- INSERT
INSERT INTO f_roles
  (_idx, id, name)
VALUES
  ('1790448435265', 'r2', 'editor')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- INSERT
INSERT INTO f_orders
  (_idx, id, user_id, _source)
VALUES
  ('1790448435266', 'o1', 'u1', '{"amount":100.5}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."user_id"', user_id
) AS result;

-- INSERT
INSERT INTO f_orders
  (_idx, id, user_id, _source)
VALUES
  ('1790448435266', 'o2', 'u1', '{"amount":200}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."user_id"', user_id
) AS result;

-- INSERT
INSERT INTO f_orders
  (_idx, id, user_id, _source)
VALUES
  ('1790448435266', 'o3', 'u3', '{"amount":50}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."user_id"', user_id
) AS result;

-- INSERT
INSERT INTO f_orders_items
  (id, order_id, product)
VALUES
  ('i1', 'o1', 'router')
RETURNING id, product;

-- INSERT
INSERT INTO f_orders_items
  (id, order_id, product)
VALUES
  ('i2', 'o1', 'cable')
RETURNING id, product;

-- INSERT
INSERT INTO f_products
  (category, id, name, _source)
VALUES
  ('internet', 'p1', 'Plan 200', '{"price":90000}')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- INSERT
INSERT INTO f_products
  (category, id, name, _source)
VALUES
  ('internet', 'p2', 'Plan 500', '{"price":150000}')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- INSERT
INSERT INTO f_products
  (category, id, name, _source)
VALUES
  ('tv', 'p3', 'Decoder', '{"price":20000}')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- INSERT
INSERT INTO f_users_f_roles
  (_idx, role_id, user_id)
VALUES
  ('1790448435267', 'r1', 'u1')
RETURNING user_id, role_id;

-- INSERT
INSERT INTO f_users_f_roles
  (_idx, role_id, user_id)
VALUES
  ('1790448435267', 'r2', 'u1')
RETURNING user_id, role_id;

-- INSERT
INSERT INTO f_users_f_roles
  (_idx, role_id, user_id)
VALUES
  ('1790448435267', 'r2', 'u2')
RETURNING user_id, role_id;

-- ========== 3. Relaciones y campos calculados · Detail en el select (DefineDetail) [pass]

-- QUERY
SELECT
json_object(
'id', A.id
) AS result
FROM f_orders AS A
WHERE A.id = 'o1'
LIMIT 1;

-- QUERY
SELECT
A.product
FROM f_orders_items AS A
WHERE A.order_id = 'o1'
LIMIT 30;

-- ========== 3. Relaciones y campos calculados · Model.Detail + Query.Detail [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."user_id"', A.user_id
) AS result
FROM f_orders AS A
WHERE A.id = 'o1'
LIMIT 1;

-- QUERY
SELECT
A.product
FROM f_orders_items AS A
WHERE A.order_id = 'o1'
LIMIT 30;

-- ========== 3. Relaciones y campos calculados · Master en el select (DefineMaster) [pass]

-- QUERY
SELECT
json_object(
'id', A.id
) AS result
FROM f_users AS A
WHERE A.id = 'u1'
LIMIT 1;

-- QUERY
SELECT
json_object(
'name', A.name
) AS result
FROM f_roles AS A
INNER JOIN f_users_f_roles AS B
  ON B.role_id = A.id
WHERE B.user_id = 'u1'
LIMIT 1000;

-- ========== 3. Relaciones y campos calculados · Master 1 a 1 (rows = 1) [pass]

-- QUERY
SELECT
json_object(
'id', A.id
) AS result
FROM f_users AS A
WHERE A.id = 'u2'
LIMIT 1;

-- QUERY
SELECT
json_object(
'name', A.name
) AS result
FROM f_roles AS A
INNER JOIN f_users_f_roles AS B
  ON B.role_id = A.id
WHERE B.user_id = 'u2'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · Model.Master + Model.Bridge [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name
) AS result
FROM f_roles AS A
INNER JOIN f_users_f_roles AS B
  ON A.id = B.role_id
WHERE B.user_id = 'u2'
LIMIT 1000;

-- ========== 3. Relaciones y campos calculados · DefineRollup row (tp_doc → title) [pass]

-- QUERY
SELECT
json_object(
'id', A.id,
'tp_doc', json(A._source -> '$."tp_doc"')
) AS result
FROM f_users AS A
ORDER BY A.id ASC
LIMIT 1000;

-- QUERY
SELECT
json_object(
'title', A.title
) AS result
FROM f_doc_types AS A
WHERE A.id = 'CC'
LIMIT 1;

-- QUERY
SELECT
json_object(
'title', A.title
) AS result
FROM f_doc_types AS A
WHERE A.id = NULL
LIMIT 1;

-- QUERY
SELECT
json_object(
'title', A.title
) AS result
FROM f_doc_types AS A
WHERE A.id = 'NIT'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · DefineRollup (count, sum, object) [pass]

-- QUERY
SELECT
json_object(
'id', A.id
) AS result
FROM f_users AS A
WHERE A.id = 'u3'
LIMIT 1;

-- QUERY
SELECT
json_object(
'amount', COALESCE(SUM(CAST(json_extract(A._source, '$."amount"') AS REAL)), 0)
) AS result
FROM f_orders AS A
WHERE A.user_id = 'u3'
LIMIT 1000;

-- QUERY
SELECT
json_object(
'id', A.id,
'amount', json(A._source -> '$."amount"')
) AS result
FROM f_orders AS A
WHERE A.user_id = 'u3'
LIMIT 1;

-- QUERY
SELECT COUNT(*) AS count
FROM f_orders AS A
WHERE A.user_id = 'u3';

-- ========== 3. Relaciones y campos calculados · DefineCalcFunc + Model.Calc [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.id = 'u1'
LIMIT 1;

-- ========== 3. Relaciones y campos calculados · DefineCalc (script JS) [pass]

-- QUERY
SELECT
json_object(
'id', A.id,
'name', A.name
) AS result
FROM f_users AS A
WHERE A.id = 'u3'
LIMIT 1;

-- ========== 4. Consultas · Select + Where + OrderBy + All [pass]

-- QUERY
SELECT
json_object(
'id', A.id,
'name', A.name,
'age', json(A._source -> '$."age"')
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) > 18
ORDER BY CAST(json_extract(A._source, '$."age"') AS INTEGER) DESC
LIMIT 1000;

-- ========== 4. Consultas · One / First / Count / Exists [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.id = 'u2'
LIMIT 1;

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
ORDER BY A.id ASC
LIMIT 2;

-- QUERY
SELECT COUNT(*) AS count
FROM f_users AS A;

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_users AS A
WHERE A.id = 'u9') AS "exists";

-- ========== 4. Consultas · Limit (paginación) / Page [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
ORDER BY A.id ASC
LIMIT 1
OFFSET 1;

-- ========== 4. Consultas · Hidden (campo oculto en la consulta y en el modelo) [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."email"', '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name
) AS result
FROM f_users AS A
WHERE A.id = 'u1'
LIMIT 1;

-- ========== 4. Consultas · Join (fluido) [pass]

-- QUERY
SELECT
json_object(
'id', A.id,
'user', U.name
) AS result
FROM f_orders AS A
INNER JOIN f_users AS U
  ON A.user_id = U.id
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 4. Consultas · LeftJoin + GroupBy [pass]

-- QUERY
SELECT
json_object(
'id', A.id,
'n', COUNT(O.id)
) AS result
FROM f_users AS A
LEFT JOIN f_orders AS O
  ON O.user_id = A.id
GROUP BY A.id
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 4. Consultas · GroupBy + Having (fluido) [pass]

-- QUERY
SELECT
json_object(
'user_id', A.user_id,
'n', COUNT(A.id),
'total', COALESCE(SUM(CAST(json_extract(A._source, '$."amount"') AS REAL)), 0)
) AS result
FROM f_orders AS A
GROUP BY A.user_id
HAVING COUNT(A.id) > 1
LIMIT 1000;

-- ========== 4. Consultas · Model.Query (JSON) [pass]

-- QUERY
SELECT
json_object(
'id', A.id,
'name', A.name
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) >= 18
ORDER BY A.name DESC
LIMIT 10;

-- ========== 4. Consultas · Test (genera el SQL sin ejecutarlo) / Debug [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.id = 'u1'
LIMIT 1000;

-- ========== 5. Condiciones · Eq [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.name = 'Ana'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Neg [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.name != 'Ana'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Less [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) < 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · LessEq [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) <= 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · More [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) > 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · MoreEq [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) >= 30
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Like [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.name LIKE '%an%'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · In [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.id IN ('u1', 'u3')
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · NotIn [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.id NOT IN ('u1', 'u3')
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Is (NULL) [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE json_extract(A._source, '$."nickname"') IS NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · IsNot (NULL) [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) IS NOT NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Null [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE json_extract(A._source, '$."nickname"') IS NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · NotNull [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.email IS NOT NULL
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Between [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) BETWEEN 18 AND 40
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · NotBetween [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) NOT BETWEEN 18 AND 40
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · Where (genérico) [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.name = 'Luis'
ORDER BY A.id ASC
LIMIT 1000;

-- ========== 5. Condiciones · And / Or (conectores) [pass]

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE A.name = 'Ana'
  OR A.name = 'Luis'
ORDER BY A.id ASC
LIMIT 1000;

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"', '$."password"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name,
'$."email"', A.email
) AS result
FROM f_users AS A
WHERE CAST(json_extract(A._source, '$."age"') AS INTEGER) > 18
  AND A.name LIKE '%neil%'
LIMIT 1000;

-- ========== 6. Comandos · Insert (RETURNING) [pass]

-- INSERT
INSERT INTO f_products
  (category, id, name, _source)
VALUES
  ('equipos', 'p4', 'Router Wi-Fi 6', '{"price":350000}')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- ========== 6. Comandos · Bulk [pass]

-- BULK
INSERT INTO f_products
  (category, id, name, _source)
VALUES
  ('tv', 'p5', 'Cable HDMI', '{"price":15000}')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- BULK
INSERT INTO f_products
  (category, id, name, _source)
VALUES
  ('tv', 'p6', 'Control', '{"price":10000}')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- ========== 6. Comandos · Update + Where [pass]

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 'p1'
LIMIT 1000;

-- UPDATE
UPDATE f_products
SET
  category = 'internet',
  name = 'Plan 200',
  _source = json_set(COALESCE(_source, '{}'),
'$."price"', json('99000'),
'$."promo"', json('true')
)
WHERE id = 'p1'
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- ========== 6. Comandos · Update + Where + Or (varias filas) [pass]

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 'p5'
  OR A.id = 'p6'
LIMIT 1000;

-- UPDATE
UPDATE f_products
SET
  category = 'tv',
  name = 'Cable HDMI',
  _source = json_set(COALESCE(_source, '{}'),
'$."price"', json('15000'),
'$."stock"', json('5')
)
WHERE id = 'p5'
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- UPDATE
UPDATE f_products
SET
  category = 'tv',
  name = 'Control',
  _source = json_set(COALESCE(_source, '{}'),
'$."price"', json('10000'),
'$."stock"', json('5')
)
WHERE id = 'p6'
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- ========== 6. Comandos · Upsert (inserta y luego actualiza) [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_products AS A
WHERE A.id = 'p7') AS "exists";

-- INSERT
INSERT INTO f_products
  (category, id, name)
VALUES
  ('tv', 'p7', 'Antena')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_products AS A
WHERE A.id = 'p7') AS "exists";

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 'p7'
LIMIT 1000;

-- UPDATE
UPDATE f_products
SET
  category = 'tv',
  name = 'Antena HD'
WHERE id = 'p7'
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- ========== 6. Comandos · Return (campos del RETURNING) [pass]

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 'p4'
LIMIT 1000;

-- UPDATE
UPDATE f_products
SET
  category = 'equipos',
  name = 'Router Wi-Fi 6',
  _source = json_set(COALESCE(_source, '{}'),
'$."price"', json('350000'),
'$."stock"', json('7')
)
WHERE id = 'p4'
RETURNING id;

-- ========== 6. Comandos · Delete (devuelve la fila borrada) [pass]

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 'p6'
LIMIT 1000;

-- DELETE
DELETE FROM f_products
WHERE id = 'p6'
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_products AS A
WHERE A.id = 'p6') AS "exists";

-- ========== 6. Comandos · Test (no ejecuta) + ToJson [pass]

-- INSERT
INSERT INTO f_products
  (id, name)
VALUES
  ('p8', 'No se guarda')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_products AS A
WHERE A.id = 'p8') AS "exists";

-- ========== 7. Triggers · Before/After Insert, Update, Delete e InsertOrUpdate [pass]

-- DDL
CREATE TABLE IF NOT EXISTS f_events (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS f_events_status_idx ON f_events (status);
CREATE INDEX IF NOT EXISTS f_events__idx_idx ON f_events (_idx);

-- INSERT
INSERT INTO f_events
  (_idx, id, name, _source)
VALUES
  ('1790448435273', 'e1', 'alta', '{"stage":"before_insert","touched":true}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name
) AS result
FROM f_events AS A
WHERE A.id = 'e1'
LIMIT 1000;

-- UPDATE
UPDATE f_events
SET
  created_at = NULL,
  name = 'cambio',
  status = 'active',
  updated_at = NULL,
  _source = json_set(COALESCE(_source, '{}'),
'$."stage"', json('"before_update"'),
'$."touched"', json('true')
)
WHERE id = 'e1'
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name
) AS result
FROM f_events AS A
WHERE A.id = 'e1'
LIMIT 1000;

-- DELETE
DELETE FROM f_events
WHERE id = 'e1'
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- ========== 7. Triggers · Trigger del comando + error que aborta [pass]

-- INSERT
INSERT INTO f_events
  (_idx, id, name, _source)
VALUES
  ('1790448435273', 'e2', 'cmd', '{"source":"command","stage":"before_insert","touched":true}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_events AS A
WHERE A.id = 'e3') AS "exists";

-- ========== 7. Triggers · Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) [pass]

-- INSERT
INSERT INTO f_events
  (_idx, id, name, _source)
VALUES
  ('1790448435273', 'e4', 'js', '{"js":"before_insert","stage":"before_insert","touched":true}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- QUERY
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."name"', A.name
) AS result
FROM f_events AS A
WHERE A.id = 'e4'
LIMIT 1000;

-- UPDATE
UPDATE f_events
SET
  created_at = NULL,
  name = 'js2',
  status = 'active',
  updated_at = NULL,
  _source = json_set(COALESCE(_source, '{}'),
'$."js"', json('"before_update"'),
'$."stage"', json('"before_update"'),
'$."touched"', json('true')
)
WHERE id = 'e4'
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- ========== 8. Transacciones · NewTx + ExecTx + Rollback [pass]

-- INSERT
INSERT INTO f_products
  (id, name)
VALUES
  ('t1', 'rollback')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- QUERY
SELECT EXISTS(SELECT 1
FROM f_products AS A
WHERE A.id = 't1') AS "exists";

-- ========== 8. Transacciones · NewTx + ExecTx + Commit [pass]

-- INSERT
INSERT INTO f_products
  (id, name)
VALUES
  ('t2', 'commit')
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 't2'
LIMIT 1000;

-- UPDATE
UPDATE f_products
SET
  category = NULL,
  name = 'commit 2'
WHERE id = 't2'
RETURNING json_set(COALESCE(_source, '{}'),
'$."id"', id,
'$."name"', name,
'$."category"', category
) AS result;

-- QUERY
SELECT
json_set(COALESCE(A._source, '{}'),
'$."id"', A.id,
'$."name"', A.name,
'$."category"', A.category
) AS result
FROM f_products AS A
WHERE A.id = 't2'
LIMIT 1;

-- ========== 9. Series · DefineSeries [pass]

-- DDL
CREATE TABLE IF NOT EXISTS series (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  tag TEXT DEFAULT NULL,
  format TEXT DEFAULT NULL,
  value INTEGER DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (tag)
);

CREATE INDEX IF NOT EXISTS series__idx_idx ON series (_idx);

-- ========== 9. Series · SetSeries + GetSeries [pass]

-- QUERY
SELECT EXISTS(SELECT 1
FROM series AS A
WHERE A.tag = 'invoice') AS "exists";

-- INSERT
INSERT INTO series
  (_idx, created_at, format, tag, updated_at, value)
VALUES
  ('1790448435275', '2026-09-26 13:47:15', 'FAC-%05d', 'invoice', '2026-09-26 13:47:15', 10)
RETURNING created_at, updated_at, tag, format, value;

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM series AS A
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
FROM series AS A
WHERE A.tag = 'invoice'
LIMIT 1000;

-- UPDATE
UPDATE series
SET
  created_at = '2026-09-26 13:47:15',
  format = 'FAC-%05d',
  updated_at = '2026-09-26 13:47:15',
  value = 11
WHERE tag = 'invoice'
RETURNING created_at, updated_at, tag, format, value;

-- QUERY
SELECT EXISTS(SELECT 1
FROM series AS A
WHERE A.tag = 'invoice') AS "exists";

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM series AS A
WHERE A.tag = 'invoice'
LIMIT 1000;

-- UPDATE
UPDATE series
SET
  created_at = '2026-09-26 13:47:15',
  format = 'FAC-%05d',
  updated_at = '2026-09-26 13:47:15',
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
FROM series AS A
WHERE A.tag = 'invoice'
LIMIT 1000;

-- DELETE
DELETE FROM series
WHERE tag = 'invoice'
RETURNING created_at, updated_at, tag, format, value;

-- QUERY
SELECT
A.created_at,
A.updated_at,
A.tag,
A.format,
A.value
FROM series AS A
WHERE A.tag = 'invoice'
LIMIT 1;


-- ========== 2. Definición de modelos (DDL) · Define (declarativo) [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_products]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_products] (
  [id] NVARCHAR(80) NOT NULL,
  [name] NVARCHAR(255) DEFAULT NULL,
  [category] NVARCHAR(80) DEFAULT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  CONSTRAINT [jsql_catalog_f_products_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_products_category_idx] ON [jsql_catalog].[f_products] ([category]);

-- ========== 2. Definición de modelos (DDL) · DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_roles]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_roles] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [name] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_roles_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_roles_status_idx] ON [jsql_catalog].[f_roles] ([status]);
CREATE INDEX [f_roles__idx_idx] ON [jsql_catalog].[f_roles] ([_idx]);

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_users]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_users] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [name] NVARCHAR(80) DEFAULT NULL,
  [email] NVARCHAR(255) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_users_pkey] PRIMARY KEY ([id])
);
CREATE UNIQUE INDEX [f_users_email_key] ON [jsql_catalog].[f_users] ([email]);
CREATE INDEX [f_users_status_idx] ON [jsql_catalog].[f_users] ([status]);
CREATE INDEX [f_users__idx_idx] ON [jsql_catalog].[f_users] ([_idx]);

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_users_f_roles]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_users_f_roles] (
  [user_id] NVARCHAR(80) NOT NULL,
  [role_id] NVARCHAR(80) NOT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_users_f_roles_pkey] PRIMARY KEY ([user_id], [role_id])
);
CREATE INDEX [f_users_f_roles__idx_idx] ON [jsql_catalog].[f_users_f_roles] ([_idx]);
ALTER TABLE [jsql_catalog].[f_users_f_roles] ADD CONSTRAINT [fk_jsql_catalog_f_users_f_roles_f_users] FOREIGN KEY ([user_id]) REFERENCES [jsql_catalog].[f_users] ([id]) ON DELETE CASCADE;
ALTER TABLE [jsql_catalog].[f_users_f_roles] ADD CONSTRAINT [fk_jsql_catalog_f_users_f_roles_f_roles] FOREIGN KEY ([role_id]) REFERENCES [jsql_catalog].[f_roles] ([id]) ON DELETE CASCADE;

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_doc_types]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_doc_types] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [title] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_doc_types_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_doc_types_status_idx] ON [jsql_catalog].[f_doc_types] ([status]);
CREATE INDEX [f_doc_types__idx_idx] ON [jsql_catalog].[f_doc_types] ([_idx]);

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_orders]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_orders] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [user_id] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_orders_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_orders_status_idx] ON [jsql_catalog].[f_orders] ([status]);
CREATE INDEX [f_orders__idx_idx] ON [jsql_catalog].[f_orders] ([_idx]);

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_orders_items]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_orders_items] (
  [order_id] NVARCHAR(80) DEFAULT NULL,
  [id] NVARCHAR(80) NOT NULL,
  [product] NVARCHAR(255) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_orders_items_pkey] PRIMARY KEY ([id])
);
ALTER TABLE [jsql_catalog].[f_orders_items] ADD CONSTRAINT [fk_jsql_catalog_f_orders_items_f_orders] FOREIGN KEY ([order_id]) REFERENCES [jsql_catalog].[f_orders] ([id]) ON DELETE CASCADE;

-- ========== 2. Definición de modelos (DDL) · DefineTenantModel / DefineProjectModel [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_tenant]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_tenant] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [tenant_id] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_tenant_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_tenant_status_idx] ON [jsql_catalog].[f_tenant] ([status]);
CREATE INDEX [f_tenant__idx_idx] ON [jsql_catalog].[f_tenant] ([_idx]);
CREATE INDEX [f_tenant_tenant_id_idx] ON [jsql_catalog].[f_tenant] ([tenant_id]);

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_project]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_project] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [project_id] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_project_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_project_status_idx] ON [jsql_catalog].[f_project] ([status]);
CREATE INDEX [f_project__idx_idx] ON [jsql_catalog].[f_project] ([_idx]);
CREATE INDEX [f_project_project_id_idx] ON [jsql_catalog].[f_project] ([project_id]);

-- ========== 2. Definición de modelos (DDL) · DefineRequired (rechaza el insert sin el campo) [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_required]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_required] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [name] NVARCHAR(255) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_required_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_required_status_idx] ON [jsql_catalog].[f_required] ([status]);
CREATE INDEX [f_required__idx_idx] ON [jsql_catalog].[f_required] ([_idx]);

-- ========== 2. Definición de modelos (DDL) · DefineForeignKeys (rechaza un hijo sin padre) [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_parent]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_parent] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_parent_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_parent_status_idx] ON [jsql_catalog].[f_parent] ([status]);
CREATE INDEX [f_parent__idx_idx] ON [jsql_catalog].[f_parent] ([_idx]);

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_child]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_child] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [parent_id] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_child_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_child_status_idx] ON [jsql_catalog].[f_child] ([status]);
CREATE INDEX [f_child__idx_idx] ON [jsql_catalog].[f_child] ([_idx]);
ALTER TABLE [jsql_catalog].[f_child] ADD CONSTRAINT [fk_jsql_catalog_f_child_f_parent] FOREIGN KEY ([parent_id]) REFERENCES [jsql_catalog].[f_parent] ([id]) ON DELETE CASCADE;

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_parent]
  ([_idx], [id])
  VALUES (N'1790553369006', N'p1');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_parent] WHERE [id] = N'p1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_child]
  ([_idx], [id], [parent_id])
  VALUES (N'1790553369032', N'c1', N'p1');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"parent_id":', CASE WHEN [parent_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([parent_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_child] WHERE [id] = N'c1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_child]
  ([_idx], [id], [parent_id])
  VALUES (N'1790553369040', N'c2', N'missing');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"parent_id":', CASE WHEN [parent_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([parent_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_child] WHERE [id] = N'c2';

-- ========== 2. Definición de modelos (DDL) · Stricted (ignora campos desconocidos) [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_strict]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_strict] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [name] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_strict_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_strict_status_idx] ON [jsql_catalog].[f_strict] ([status]);
CREATE INDEX [f_strict__idx_idx] ON [jsql_catalog].[f_strict] ([_idx]);

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_strict]
  ([_idx], [id], [name])
  VALUES (N'1790553369102', N's1', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_strict] WHERE [id] = N's1';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_strict] AS A
WHERE A.[id] = N's1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 2. Definición de modelos (DDL) · DB.Query define (Define en JSON) [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_json_model]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_json_model] (
  [id] NVARCHAR(80) NOT NULL,
  [title] NVARCHAR(255) DEFAULT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  CONSTRAINT [jsql_catalog_f_json_model_pkey] PRIMARY KEY ([id])
);

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_json_model]
  ([id], [title], [_source])
  VALUES (N'j1', N'desde define', N'{"extra":1}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"title":', CASE WHEN [title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_json_model] WHERE [id] = N'j1';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"title":', CASE WHEN A.[title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_json_model] AS A
WHERE A.[id] = N'j1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 2. Definición de modelos (DDL) · Insert (datos base) [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_doc_types]
  ([_idx], [id], [title])
  VALUES (N'1790553369138', N'CC', N'Cédula de ciudadanía');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"title":', CASE WHEN [title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_doc_types] WHERE [id] = N'CC';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_doc_types]
  ([_idx], [id], [title])
  VALUES (N'1790553369144', N'NIT', N'Número de identificación tributaria');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"title":', CASE WHEN [title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_doc_types] WHERE [id] = N'NIT';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'ana@example.com') THEN 'true' ELSE 'false' END AS [exists]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users]
  ([_idx], [email], [id], [name], [_source])
  VALUES (N'1790553369154', N'ana@example.com', N'u1', N'Ana', N'{"age":30,"password":"secret","tp_doc":"CC"}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN [email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users] WHERE [id] = N'u1';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'luis@example.com') THEN 'true' ELSE 'false' END AS [exists]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users]
  ([_idx], [email], [id], [name], [_source])
  VALUES (N'1790553369165', N'luis@example.com', N'u2', N'Luis', N'{"age":17}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN [email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users] WHERE [id] = N'u2';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'marta@example.com') THEN 'true' ELSE 'false' END AS [exists]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users]
  ([_idx], [email], [id], [name], [_source])
  VALUES (N'1790553369173', N'marta@example.com', N'u3', N'Marta O''Neil', N'{"age":45,"tp_doc":"NIT"}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN [email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users] WHERE [id] = N'u3';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_roles]
  ([_idx], [id], [name])
  VALUES (N'1790553369177', N'r1', N'admin');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_roles] WHERE [id] = N'r1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_roles]
  ([_idx], [id], [name])
  VALUES (N'1790553369185', N'r2', N'editor');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_roles] WHERE [id] = N'r2';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_orders]
  ([_idx], [id], [user_id], [_source])
  VALUES (N'1790553369191', N'o1', N'u1', N'{"amount":100.5}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"user_id":', CASE WHEN [user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([user_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_orders] WHERE [id] = N'o1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_orders]
  ([_idx], [id], [user_id], [_source])
  VALUES (N'1790553369198', N'o2', N'u1', N'{"amount":200}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"user_id":', CASE WHEN [user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([user_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_orders] WHERE [id] = N'o2';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_orders]
  ([_idx], [id], [user_id], [_source])
  VALUES (N'1790553369204', N'o3', N'u3', N'{"amount":50}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"user_id":', CASE WHEN [user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([user_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_orders] WHERE [id] = N'o3';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_orders_items]
  ([id], [order_id], [product])
  VALUES (N'i1', N'o1', N'router');
SELECT CONCAT('{', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"product":', CASE WHEN [product] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([product] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_orders_items] WHERE [id] = N'i1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_orders_items]
  ([id], [order_id], [product])
  VALUES (N'i2', N'o1', N'cable');
SELECT CONCAT('{', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"product":', CASE WHEN [product] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([product] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_orders_items] WHERE [id] = N'i2';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'internet', N'p1', N'Plan 200', N'{"price":90000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'internet', N'p2', N'Plan 500', N'{"price":150000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p2';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'tv', N'p3', N'Decoder', N'{"price":20000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p3';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users_f_roles]
  ([_idx], [role_id], [user_id])
  VALUES (N'1790553369239', N'r1', N'u1');
SELECT CONCAT('{', N'"user_id":', CASE WHEN [user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([user_id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"role_id":', CASE WHEN [role_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([role_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users_f_roles] WHERE [user_id] = N'u1' AND [role_id] = N'r1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users_f_roles]
  ([_idx], [role_id], [user_id])
  VALUES (N'1790553369245', N'r2', N'u1');
SELECT CONCAT('{', N'"user_id":', CASE WHEN [user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([user_id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"role_id":', CASE WHEN [role_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([role_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users_f_roles] WHERE [user_id] = N'u1' AND [role_id] = N'r2';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users_f_roles]
  ([_idx], [role_id], [user_id])
  VALUES (N'1790553369249', N'r2', N'u2');
SELECT CONCAT('{', N'"user_id":', CASE WHEN [user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([user_id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"role_id":', CASE WHEN [role_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([role_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users_f_roles] WHERE [user_id] = N'u2' AND [role_id] = N'r2';

-- ========== 2. Definición de modelos (DDL) · DefineUnique (rechaza duplicados en insert, bulk y update) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'ana@example.com') THEN 'true' ELSE 'false' END AS [exists]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'same@example.com') THEN 'true' ELSE 'false' END AS [exists]

-- BULK
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_users]
  ([_idx], [email], [id], [name])
  VALUES (N'1790553369320', N'same@example.com', N'u8', N'Uno');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN [email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users] WHERE [id] = N'u8';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'same@example.com') THEN 'true' ELSE 'false' END AS [exists]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] = N'ana@example.com'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 2 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_users]
  SET [created_at] = NULL,
    [email] = N'luis@example.com',
    [name] = N'Luis',
    [status] = N'active',
    [updated_at] = NULL,
    [_source] = JSON_MODIFY(COALESCE([_source], N'{}'), N'$."age"', 17)
  WHERE [id] = N'u2';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN [email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_users] WHERE [id] = N'u2';

-- QUERY
SELECT COUNT(*) AS [count]
FROM [jsql_catalog].[f_users] AS A

-- ========== 3. Relaciones y campos calculados · Detail en el select (DefineDetail) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
WHERE A.[id] = N'o1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"product":', CASE WHEN A.[product] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[product] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_orders_items] AS A
WHERE A.[order_id] = N'o1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 30 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Model.Detail + Query.Detail [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"user_id":', CASE WHEN A.[user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[user_id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
WHERE A.[id] = N'o1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"product":', CASE WHEN A.[product] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[product] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_orders_items] AS A
WHERE A.[order_id] = N'o1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 30 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Master en el select (DefineMaster) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_roles] AS A
INNER JOIN [jsql_catalog].[f_users_f_roles] AS B
  ON B.[role_id] = A.[id]
WHERE B.[user_id] = N'u1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Master 1 a 1 (rows = 1) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_roles] AS A
INNER JOIN [jsql_catalog].[f_users_f_roles] AS B
  ON B.[role_id] = A.[id]
WHERE B.[user_id] = N'u2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Model.Master + Model.Bridge [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_roles] AS A
INNER JOIN [jsql_catalog].[f_users_f_roles] AS B
  ON A.[id] = B.[role_id]
WHERE B.[user_id] = N'u2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineRollup row (tp_doc → title) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"tp_doc":', COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('"', STRING_ESCAPE([value], 'json'), '"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON(A.[_source], N'$') WHERE [key] = N'tp_doc'), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"title":', CASE WHEN A.[title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_doc_types] AS A
WHERE A.[id] = N'CC'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"title":', CASE WHEN A.[title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_doc_types] AS A
WHERE A.[id] = NULL
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"title":', CASE WHEN A.[title] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[title] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_doc_types] AS A
WHERE A.[id] = N'NIT'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineRollup (count, sum, object) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u3'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT COUNT(*) AS [count]
FROM [jsql_catalog].[f_orders] AS A
WHERE A.[user_id] = N'u3'

-- QUERY
SELECT
CONCAT('{', N'"amount":', COALESCE(CONVERT(NVARCHAR(60), COALESCE(SUM(TRY_CAST(JSON_VALUE(A.[_source], N'$."amount"') AS DECIMAL(38,10))), 0)), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
WHERE A.[user_id] = N'u3'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"amount":', COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('"', STRING_ESCAPE([value], 'json'), '"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON(A.[_source], N'$') WHERE [key] = N'amount'), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
WHERE A.[user_id] = N'u3'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineCalcFunc + Model.Calc [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Query.Calc con script JS (DefineCalc) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u3'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineCalc (script JS) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u3'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Select + Where + OrderBy + All [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"age":', COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('"', STRING_ESCAPE([value], 'json'), '"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON(A.[_source], N'$') WHERE [key] = N'age'), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) > 18
ORDER BY TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) DESC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · One / First / Count / Exists [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 2 ROWS ONLY

-- QUERY
SELECT COUNT(*) AS [count]
FROM [jsql_catalog].[f_users] AS A

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u9') THEN 'true' ELSE 'false' END AS [exists]

-- ========== 4. Consultas · Limit (paginación) / Page [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
ORDER BY A.[id] ASC
OFFSET 1 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Hidden (campo oculto en la consulta y en el modelo) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."email"', NULL), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."email"', NULL), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."email"', NULL), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Hidden pedido por nombre (se devuelve) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"password":', COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('"', STRING_ESCAPE([value], 'json'), '"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON(A.[_source], N'$') WHERE [key] = N'password'), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Join (fluido) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"user":', CASE WHEN U.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
INNER JOIN [jsql_catalog].[f_users] AS U
  ON A.[user_id] = U.[id]
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · LeftJoin + GroupBy [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(O.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
LEFT JOIN [jsql_catalog].[f_orders] AS O
  ON O.[user_id] = A.[id]
GROUP BY A.[id]
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · RightJoin / FullJoin [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN U.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(A.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
RIGHT JOIN [jsql_catalog].[f_users] AS U
  ON A.[user_id] = U.[id]
GROUP BY U.[id]
ORDER BY U.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(O.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
FULL JOIN [jsql_catalog].[f_orders] AS O
  ON O.[user_id] = A.[id]
GROUP BY A.[id]
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · GroupBy + Having (fluido) [pass]

-- QUERY
SELECT
CONCAT('{', N'"user_id":', CASE WHEN A.[user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[user_id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(A.[id])), 'null'), ',', N'"total":', COALESCE(CONVERT(NVARCHAR(60), COALESCE(SUM(TRY_CAST(JSON_VALUE(A.[_source], N'$."amount"') AS DECIMAL(38,10))), 0)), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
GROUP BY A.[user_id]
HAVING COUNT(A.[id]) > 1
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · OrderBy por alias de un agregado [pass]

-- QUERY
SELECT
CONCAT('{', N'"user_id":', CASE WHEN A.[user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[user_id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(A.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
GROUP BY A.[user_id]
ORDER BY COUNT(A.[id]) DESC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query (JSON) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) >= 18
ORDER BY A.[name] DESC
OFFSET 0 ROWS FETCH NEXT 10 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from (otro modelo y alias) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN U.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN U.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS U
WHERE TRY_CAST(JSON_VALUE(U.[_source], N'$."age"') AS BIGINT) > 18
ORDER BY U.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from de varios orígenes [pass]

-- QUERY
SELECT
CONCAT('{', N'"name":', CASE WHEN U.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"order":', CASE WHEN O.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(O.[id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS U,
[jsql_catalog].[f_orders] AS O
WHERE O.[user_id] = U.[id]
ORDER BY O.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from sin esquema [pass]

-- QUERY
SELECT
CONCAT('{', N'"name":', CASE WHEN R.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(R.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_roles] AS R
ORDER BY R.[name] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from + join + groups [pass]

-- QUERY
SELECT
CONCAT('{', N'"name":', CASE WHEN U.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(O.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS U
INNER JOIN [jsql_catalog].[f_orders] AS O
  ON O.[user_id] = U.[id]
GROUP BY U.[name]
ORDER BY U.[name] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · DB.Query consulta (select, from, join, group by, order by) [pass]

-- QUERY
SELECT
CONCAT('{', N'"name":', CASE WHEN U.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(U.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(O.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_users] AS U
LEFT JOIN [jsql_catalog].[f_orders] AS O
  ON O.[user_id] = U.[id]
GROUP BY U.[name]
ORDER BY U.[name] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · DB.Query consulta (where + and/or de primer nivel, limit, offset) [pass]

-- QUERY
SELECT
CONCAT('{', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) > 18
  OR A.[name] = N'Luis'
ORDER BY A.[id] ASC
OFFSET 1 ROWS FETCH NEXT 2 ROWS ONLY

-- ========== 4. Consultas · Model.Query con claves select / group by / order by [pass]

-- QUERY
SELECT
CONCAT('{', N'"user_id":', CASE WHEN A.[user_id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[user_id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"n":', COALESCE(CONVERT(NVARCHAR(40), COUNT(A.[id])), 'null'), '}') AS [result]
FROM [jsql_catalog].[f_orders] AS A
GROUP BY A.[user_id]
HAVING COUNT(A.[id]) > 1
ORDER BY A.[user_id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Test (genera el SQL sin ejecutarlo) / Debug [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] = N'u1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Eq [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[name] = N'Ana'
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Neg [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[name] != N'Ana'
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Less [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) < 30
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · LessEq [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) <= 30
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · More [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) > 30
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · MoreEq [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) >= 30
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Like [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE LOWER(A.[name]) LIKE LOWER(N'%an%')
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · In [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] IN (N'u1', N'u3')
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · NotIn [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[id] NOT IN (N'u1', N'u3')
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Is (NULL) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE JSON_VALUE(A.[_source], N'$."nickname"') IS NULL
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · IsNot (NULL) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) IS NOT NULL
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Null [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE JSON_VALUE(A.[_source], N'$."nickname"') IS NULL
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · NotNull [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[email] IS NOT NULL
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Between [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) BETWEEN 18 AND 40
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · NotBetween [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) NOT BETWEEN 18 AND 40
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Where (genérico) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[name] = N'Luis'
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Is / IsNot con valor (comparación NULL-safe) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[name] IS NOT DISTINCT FROM N'Ana'
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[name] IS DISTINCT FROM N'Ana'
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · And / Or (conectores) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE A.[name] = N'Ana'
  OR A.[name] = N'Luis'
ORDER BY A.[id] ASC
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL), 2, LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), N'$."password"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"email":', CASE WHEN A.[email] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[email] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_users] AS A
WHERE TRY_CAST(JSON_VALUE(A.[_source], N'$."age"') AS BIGINT) > 18
  AND LOWER(A.[name]) LIKE LOWER(N'%neil%')
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- ========== 6. Comandos · Insert (RETURNING) [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'equipos', N'p4', N'Router Wi-Fi 6', N'{"price":350000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p4';

-- ========== 6. Comandos · Bulk [pass]

-- BULK
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'tv', N'p5', N'Cable HDMI', N'{"price":15000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p5';

-- BULK
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'tv', N'p6', N'Control', N'{"price":10000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p6';

-- ========== 6. Comandos · Update + Where [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'internet',
    [name] = N'Plan 200',
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."price"', 99000), N'$."promo"', CAST(1 AS BIT))
  WHERE [id] = N'p1';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p1';

-- ========== 6. Comandos · Update + Where + Or (varias filas) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p5'
  OR A.[id] = N'p6'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'tv',
    [name] = N'Cable HDMI',
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."price"', 15000), N'$."stock"', 5)
  WHERE [id] = N'p5';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p5';

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'tv',
    [name] = N'Control',
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."price"', 10000), N'$."stock"', 5)
  WHERE [id] = N'p6';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p6';

-- ========== 6. Comandos · Upsert (inserta y luego actualiza) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p7') THEN 'true' ELSE 'false' END AS [exists]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name])
  VALUES (N'tv', N'p7', N'Antena');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p7';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p7') THEN 'true' ELSE 'false' END AS [exists]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p7'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'tv',
    [name] = N'Antena HD'
  WHERE [id] = N'p7';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p7';

-- ========== 6. Comandos · Return (campos como en Select: columna, atributo, alias, ruta) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p4'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'equipos',
    [name] = N'Router Wi-Fi 6',
    [_source] = JSON_MODIFY(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."price"', 350000), N'$."specs"', JSON_QUERY(N'{"wifi":"6"}')), N'$."stock"', 7)
  WHERE [id] = N'p4';
SELECT CONCAT('{', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"nombre":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"stock":', COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('"', STRING_ESCAPE([value], 'json'), '"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON([_source], N'$') WHERE [key] = N'stock'), 'null'), ',', N'"wifi":', COALESCE((SELECT TOP 1 CASE [type] WHEN 1 THEN CONCAT('"', STRING_ESCAPE([value], 'json'), '"') WHEN 0 THEN 'null' ELSE [value] END FROM OPENJSON([_source], N'$."specs"') WHERE [key] = N'wifi'), 'null'), '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p4';

-- ========== 6. Comandos · Delete (devuelve la fila borrada) [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p6'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p6';
DELETE FROM [jsql_catalog].[f_products] WHERE [id] = N'p6';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p6') THEN 'true' ELSE 'false' END AS [exists]

-- ========== 6. Comandos · DB.Query insert + bulk (con triggers JS) [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name], [_source])
  VALUES (N'equipos', N'j1', N'Mesh', N'{"origin":"json","price":450000}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j1';

-- BULK
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name])
  VALUES (N'equipos', N'j2', N'Extensor');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j2';

-- BULK
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([category], [id], [name])
  VALUES (N'equipos', N'j3', N'Splitter');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j3';

-- ========== 6. Comandos · DB.Query update + delete [pass]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[category] = N'equipos'
  AND A.[id] IN (N'j2', N'j3')
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'equipos',
    [name] = N'Extensor',
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."price"', 99), N'$."touched"', CAST(1 AS BIT))
  WHERE [id] = N'j2';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j2';

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = N'equipos',
    [name] = N'Splitter',
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."price"', 99), N'$."touched"', CAST(1 AS BIT))
  WHERE [id] = N'j3';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j3';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'j3'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j3';
DELETE FROM [jsql_catalog].[f_products] WHERE [id] = N'j3';

-- ========== 6. Comandos · DB.Query upsert (inserta, actualiza y exige where) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'j4') THEN 'true' ELSE 'false' END AS [exists]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([id], [name], [_source])
  VALUES (N'j4', N'Repetidor', N'{"both":true,"path":"insert"}');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j4';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'j4') THEN 'true' ELSE 'false' END AS [exists]

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'j4'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = NULL,
    [name] = N'Repetidor Pro',
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."both"', CAST(1 AS BIT)), N'$."path"', N'update')
  WHERE [id] = N'j4';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'j4';

-- ========== 6. Comandos · Update / Delete con limit (por defecto, n y 0 = todas) [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_many]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_many] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [name] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_many_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_many_status_idx] ON [jsql_catalog].[f_many] ([status]);
CREATE INDEX [f_many__idx_idx] ON [jsql_catalog].[f_many] ([_idx]);

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_many]
  ([_idx], [id], [name])
  VALUES (N'1790553369790', N'm1', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm1';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_many]
  ([_idx], [id], [name])
  VALUES (N'1790553369796', N'm2', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm2';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_many]
  ([_idx], [id], [name])
  VALUES (N'1790553369802', N'm3', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm3';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_many]
  ([_idx], [id], [name])
  VALUES (N'1790553369806', N'm4', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm4';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_many]
  ([_idx], [id], [name])
  VALUES (N'1790553369811', N'm5', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm5';

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_many]
  ([_idx], [id], [name])
  VALUES (N'1790553369816', N'm6', N'x');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm6';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_many] AS A
WHERE A.[name] = N'x'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 2 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_many]
  SET [created_at] = NULL,
    [name] = N'y',
    [status] = N'active',
    [updated_at] = NULL
  WHERE [id] = N'm1';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm1';

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_many]
  SET [created_at] = NULL,
    [name] = N'y',
    [status] = N'active',
    [updated_at] = NULL
  WHERE [id] = N'm2';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm2';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_many] AS A
WHERE A.[name] = N'x'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 3 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_many]
  SET [created_at] = NULL,
    [name] = N'y',
    [status] = N'active',
    [updated_at] = NULL
  WHERE [id] = N'm3';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm3';

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_many]
  SET [created_at] = NULL,
    [name] = N'y',
    [status] = N'active',
    [updated_at] = NULL
  WHERE [id] = N'm4';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm4';

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_many]
  SET [created_at] = NULL,
    [name] = N'y',
    [status] = N'active',
    [updated_at] = NULL
  WHERE [id] = N'm5';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm5';

-- QUERY
SELECT COUNT(*) AS [count]
FROM [jsql_catalog].[f_many] AS A
WHERE A.[name] = N'x'

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_many] AS A
WHERE A.[name] IN (N'x', N'y')

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm1';
DELETE FROM [jsql_catalog].[f_many] WHERE [id] = N'm1';

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm2';
DELETE FROM [jsql_catalog].[f_many] WHERE [id] = N'm2';

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm3';
DELETE FROM [jsql_catalog].[f_many] WHERE [id] = N'm3';

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm4';
DELETE FROM [jsql_catalog].[f_many] WHERE [id] = N'm4';

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm5';
DELETE FROM [jsql_catalog].[f_many] WHERE [id] = N'm5';

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_many] WHERE [id] = N'm6';
DELETE FROM [jsql_catalog].[f_many] WHERE [id] = N'm6';

-- QUERY
SELECT COUNT(*) AS [count]
FROM [jsql_catalog].[f_many] AS A

-- ========== 6. Comandos · Upsert fluido sin where (devuelve error) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p9') THEN 'true' ELSE 'false' END AS [exists]

-- ========== 6. Comandos · Test (no ejecuta) + ToJson [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([id], [name])
  VALUES (N'p8', N'No se guarda');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N'p8';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N'p8') THEN 'true' ELSE 'false' END AS [exists]

-- ========== 7. Triggers · Before/After Insert, Update, Delete e InsertOrUpdate [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[f_events]', N'U') IS NULL CREATE TABLE [jsql_catalog].[f_events] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [status] NVARCHAR(255) DEFAULT N'active',
  [id] NVARCHAR(80) NOT NULL,
  [_source] NVARCHAR(MAX) DEFAULT N'{}' CHECK (ISJSON([_source]) = 1),
  [name] NVARCHAR(80) DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_f_events_pkey] PRIMARY KEY ([id])
);
CREATE INDEX [f_events_status_idx] ON [jsql_catalog].[f_events] ([status]);
CREATE INDEX [f_events__idx_idx] ON [jsql_catalog].[f_events] ([_idx]);

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_events]
  ([_idx], [id], [name], [_source])
  VALUES (N'1790553369913', N'e1', N'alta', N'{"stage":"before_insert","touched":true}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_events] WHERE [id] = N'e1';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_events] AS A
WHERE A.[id] = N'e1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_events]
  SET [created_at] = NULL,
    [name] = N'cambio',
    [status] = N'active',
    [updated_at] = NULL,
    [_source] = JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."stage"', N'before_update'), N'$."touched"', CAST(1 AS BIT))
  WHERE [id] = N'e1';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_events] WHERE [id] = N'e1';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_events] AS A
WHERE A.[id] = N'e1'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_events] WHERE [id] = N'e1';
DELETE FROM [jsql_catalog].[f_events] WHERE [id] = N'e1';

-- ========== 7. Triggers · Trigger del comando + error que aborta [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_events]
  ([_idx], [id], [name], [_source])
  VALUES (N'1790553369941', N'e2', N'cmd', N'{"source":"command","stage":"before_insert","touched":true}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_events] WHERE [id] = N'e2';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_events] AS A
WHERE A.[id] = N'e3') THEN 'true' ELSE 'false' END AS [exists]

-- ========== 7. Triggers · Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_events]
  ([_idx], [id], [name], [_source])
  VALUES (N'1790553369947', N'e4', N'js', N'{"js":"before_insert","stage":"before_insert","touched":true}');
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_events] WHERE [id] = N'e4';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE(A.[_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"status":', CASE WHEN A.[status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_events] AS A
WHERE A.[id] = N'e4'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_events]
  SET [created_at] = NULL,
    [name] = N'js2',
    [status] = N'active',
    [updated_at] = NULL,
    [_source] = JSON_MODIFY(JSON_MODIFY(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."js"', N'before_update'), N'$."stage"', N'before_update'), N'$."touched"', CAST(1 AS BIT))
  WHERE [id] = N'e4';
SELECT CONCAT('{', SUBSTRING(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL), 2, LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) - 2), CASE WHEN LEN(JSON_MODIFY(COALESCE([_source], N'{}'), N'$."_idx"', NULL)) > 2 THEN ',' ELSE '' END, N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"status":', CASE WHEN [status] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([status] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_events] WHERE [id] = N'e4';

-- ========== 8. Transacciones · NewTx + ExecTx + Rollback [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([id], [name])
  VALUES (N't1', N'rollback');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N't1';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N't1') THEN 'true' ELSE 'false' END AS [exists]

-- ========== 8. Transacciones · NewTx + ExecTx + Commit [pass]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[f_products]
  ([id], [name])
  VALUES (N't2', N'commit');
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N't2';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N't2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[f_products]
  SET [category] = NULL,
    [name] = N'commit 2'
  WHERE [id] = N't2';
SELECT CONCAT('{', SUBSTRING(COALESCE([_source], N'{}'), 2, LEN(COALESCE([_source], N'{}')) - 2), CASE WHEN LEN(COALESCE([_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN [id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN [name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN [category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result] FROM [jsql_catalog].[f_products] WHERE [id] = N't2';

-- QUERY
SELECT
CONCAT('{', SUBSTRING(COALESCE(A.[_source], N'{}'), 2, LEN(COALESCE(A.[_source], N'{}')) - 2), CASE WHEN LEN(COALESCE(A.[_source], N'{}')) > 2 THEN ',' ELSE '' END, N'"id":', CASE WHEN A.[id] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[id] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"name":', CASE WHEN A.[name] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[name] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"category":', CASE WHEN A.[category] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[category] AS NVARCHAR(MAX)), 'json'), '"') END, '}') AS [result]
FROM [jsql_catalog].[f_products] AS A
WHERE A.[id] = N't2'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 9. Series · DefineSeries [pass]

-- DDL
IF SCHEMA_ID(N'jsql_catalog') IS NULL EXEC(N'CREATE SCHEMA [jsql_catalog]');
IF OBJECT_ID(N'[jsql_catalog].[series]', N'U') IS NULL CREATE TABLE [jsql_catalog].[series] (
  [created_at] DATETIME2 DEFAULT NULL,
  [updated_at] DATETIME2 DEFAULT NULL,
  [tag] NVARCHAR(80) NOT NULL,
  [format] NVARCHAR(255) DEFAULT NULL,
  [value] BIGINT DEFAULT NULL,
  [_idx] NVARCHAR(80) DEFAULT NULL,
  CONSTRAINT [jsql_catalog_series_pkey] PRIMARY KEY ([tag])
);
CREATE INDEX [series__idx_idx] ON [jsql_catalog].[series] ([_idx]);

-- ========== 9. Series · SetSeries + GetSeries [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice') THEN 'true' ELSE 'false' END AS [exists]

-- INSERT
SET NOCOUNT ON;
INSERT INTO [jsql_catalog].[series]
  ([_idx], [created_at], [format], [tag], [updated_at], [value])
  VALUES (N'1790553369997', '2026-09-27T18:56:09.9979880', N'FAC-%05d', N'invoice', '2026-09-27T18:56:09.9979880', 10);
SELECT CONCAT('{', N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN [tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN [format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), [value]), 'null'), '}') AS [result] FROM [jsql_catalog].[series] WHERE [tag] = N'invoice';

-- QUERY
SELECT
CONCAT('{', N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN A.[tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN A.[format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), A.[value]), 'null'), '}') AS [result]
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY

-- ========== 9. Series · GenValue + GenSerie [pass]

-- QUERY
SELECT
CONCAT('{', N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN A.[tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN A.[format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), A.[value]), 'null'), '}') AS [result]
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[series]
  SET [created_at] = '2026-09-27T18:56:09.9979880',
    [format] = N'FAC-%05d',
    [updated_at] = '2026-09-27T18:56:10.0135130',
    [value] = 11
  WHERE [tag] = N'invoice';
SELECT CONCAT('{', N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN [tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN [format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), [value]), 'null'), '}') AS [result] FROM [jsql_catalog].[series] WHERE [tag] = N'invoice';

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice') THEN 'true' ELSE 'false' END AS [exists]

-- QUERY
SELECT
CONCAT('{', N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN A.[tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN A.[format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), A.[value]), 'null'), '}') AS [result]
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- UPDATE
SET NOCOUNT ON;
UPDATE [jsql_catalog].[series]
  SET [created_at] = '2026-09-27T18:56:09.9979880',
    [format] = N'FAC-%05d',
    [updated_at] = '2026-09-27T18:56:10.0189310',
    [value] = 12
  WHERE [tag] = N'invoice';
SELECT CONCAT('{', N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN [tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN [format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), [value]), 'null'), '}') AS [result] FROM [jsql_catalog].[series] WHERE [tag] = N'invoice';

-- ========== 9. Series · DeleteSeries [pass]

-- QUERY
SELECT
CONCAT('{', N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN A.[tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN A.[format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), A.[value]), 'null'), '}') AS [result]
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1000 ROWS ONLY

-- DELETE
SET NOCOUNT ON;
SELECT CONCAT('{', N'"created_at":', CASE WHEN [created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN [updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), [updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN [tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN [format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST([format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), [value]), 'null'), '}') AS [result] FROM [jsql_catalog].[series] WHERE [tag] = N'invoice';
DELETE FROM [jsql_catalog].[series] WHERE [tag] = N'invoice';

-- QUERY
SELECT
CONCAT('{', N'"created_at":', CASE WHEN A.[created_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[created_at], 127), '"') END, ',', N'"updated_at":', CASE WHEN A.[updated_at] IS NULL THEN 'null' ELSE CONCAT('"', CONVERT(NVARCHAR(40), A.[updated_at], 127), '"') END, ',', N'"tag":', CASE WHEN A.[tag] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[tag] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"format":', CASE WHEN A.[format] IS NULL THEN 'null' ELSE CONCAT('"', STRING_ESCAPE(CAST(A.[format] AS NVARCHAR(MAX)), 'json'), '"') END, ',', N'"value":', COALESCE(CONVERT(NVARCHAR(40), A.[value]), 'null'), '}') AS [result]
FROM [jsql_catalog].[series] AS A
WHERE A.[tag] = N'invoice'
ORDER BY (SELECT NULL)
OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY


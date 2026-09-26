-- ========== 2. Definición de modelos (DDL) · Define (declarativo) [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_products" (
  "id" VARCHAR2(80) DEFAULT NULL,
  "name" VARCHAR2(255) DEFAULT NULL,
  "category" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON)
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_products" ADD CONSTRAINT "jsql_f_products_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_products_category_idx" ON JSQL."f_products" ("category")';
END;

-- ========== 2. Definición de modelos (DDL) · DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_roles" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_roles" ADD CONSTRAINT "jsql_f_roles_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_roles_status_idx" ON JSQL."f_roles" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_roles__idx_idx" ON JSQL."f_roles" ("_idx")';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_users" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(80) DEFAULT NULL,
  "email" VARCHAR2(255) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_users" ADD CONSTRAINT "jsql_f_users_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE UNIQUE INDEX "jsql_f_users_email_key" ON JSQL."f_users" ("email")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_users_status_idx" ON JSQL."f_users" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_users__idx_idx" ON JSQL."f_users" ("_idx")';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_users_f_roles" (
  "user_id" VARCHAR2(80) DEFAULT NULL,
  "role_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_users_f_roles" ADD CONSTRAINT "jsql_f_users_f_roles_pkey" PRIMARY KEY ("user_id", "role_id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_users_f_roles__idx_idx" ON JSQL."f_users_f_roles" ("_idx")';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_users_f_roles" ADD CONSTRAINT "fk_jsql_f_users_f_roles_jsql_f_users" FOREIGN KEY ("user_id") REFERENCES JSQL."f_users" ("id") ON DELETE CASCADE';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_users_f_roles" ADD CONSTRAINT "fk_jsql_f_users_f_roles_jsql_f_roles" FOREIGN KEY ("role_id") REFERENCES JSQL."f_roles" ("id") ON DELETE CASCADE';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_doc_types" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "title" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_doc_types" ADD CONSTRAINT "jsql_f_doc_types_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_doc_types_status_idx" ON JSQL."f_doc_types" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_doc_types__idx_idx" ON JSQL."f_doc_types" ("_idx")';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_orders" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "user_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_orders" ADD CONSTRAINT "jsql_f_orders_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_orders_status_idx" ON JSQL."f_orders" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_orders__idx_idx" ON JSQL."f_orders" ("_idx")';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_orders_items" (
  "order_id" VARCHAR2(80) DEFAULT NULL,
  "id" VARCHAR2(80) DEFAULT NULL,
  "product" VARCHAR2(255) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_orders_items" ADD CONSTRAINT "jsql_f_orders_items_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_orders_items" ADD CONSTRAINT "fk_jsql_f_orders_items_jsql_f_orders" FOREIGN KEY ("order_id") REFERENCES JSQL."f_orders" ("id") ON DELETE CASCADE';
END;

-- ========== 2. Definición de modelos (DDL) · DefineTenantModel / DefineProjectModel [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_tenant" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "tenant_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_tenant" ADD CONSTRAINT "jsql_f_tenant_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_tenant_status_idx" ON JSQL."f_tenant" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_tenant__idx_idx" ON JSQL."f_tenant" ("_idx")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_tenant_tenant_id_idx" ON JSQL."f_tenant" ("tenant_id")';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_project" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "project_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_project" ADD CONSTRAINT "jsql_f_project_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_project_status_idx" ON JSQL."f_project" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_project__idx_idx" ON JSQL."f_project" ("_idx")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_project_project_id_idx" ON JSQL."f_project" ("project_id")';
END;

-- ========== 2. Definición de modelos (DDL) · DefineRequired (rechaza el insert sin el campo) [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_required" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(255) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_required" ADD CONSTRAINT "jsql_f_required_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_required_status_idx" ON JSQL."f_required" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_required__idx_idx" ON JSQL."f_required" ("_idx")';
END;

-- ========== 2. Definición de modelos (DDL) · DefineForeignKeys (rechaza un hijo sin padre) [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_parent" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_parent" ADD CONSTRAINT "jsql_f_parent_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_parent_status_idx" ON JSQL."f_parent" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_parent__idx_idx" ON JSQL."f_parent" ("_idx")';
END;

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_child" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "parent_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_child" ADD CONSTRAINT "jsql_f_child_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_child_status_idx" ON JSQL."f_child" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_child__idx_idx" ON JSQL."f_child" ("_idx")';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_child" ADD CONSTRAINT "fk_jsql_f_child_jsql_f_parent" FOREIGN KEY ("parent_id") REFERENCES JSQL."f_parent" ("id") ON DELETE CASCADE';
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_parent"
  ("_idx", "id")
  VALUES ('1790448836188', 'p1')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_parent" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_child"
  ("_idx", "id", "parent_id")
  VALUES ('1790448836267', 'c1', 'p1')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'parent_id' VALUE "parent_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_child" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_child"
  ("_idx", "id", "parent_id")
  VALUES ('1790448836284', 'c2', 'missing')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'parent_id' VALUE "parent_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_child" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 2. Definición de modelos (DDL) · Stricted (ignora campos desconocidos) [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_strict" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_strict" ADD CONSTRAINT "jsql_f_strict_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_strict_status_idx" ON JSQL."f_strict" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_strict__idx_idx" ON JSQL."f_strict" ("_idx")';
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_strict"
  ("_idx", "id", "name")
  VALUES ('1790448836311', 's1', 'x')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_strict" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_strict" A
WHERE A."id" = 's1'
FETCH NEXT 1 ROWS ONLY

-- ========== 2. Definición de modelos (DDL) · Insert (datos base) [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_doc_types"
  ("_idx", "id", "title")
  VALUES ('1790448836332', 'CC', 'Cédula de ciudadanía')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'title' VALUE "title"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_doc_types" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_doc_types"
  ("_idx", "id", "title")
  VALUES ('1790448836425', 'NIT', 'Número de identificación tributaria')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'title' VALUE "title"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_doc_types" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_users"
  ("_idx", "email", "id", "name", "_source")
  VALUES ('1790448836429', 'ana@example.com', 'u1', 'Ana', TO_CLOB('{"age":30,"password":"secret","tp_doc":"CC"}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name",
'email' VALUE "email"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_users" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_users"
  ("_idx", "email", "id", "name", "_source")
  VALUES ('1790448836464', 'luis@example.com', 'u2', 'Luis', TO_CLOB('{"age":17}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name",
'email' VALUE "email"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_users" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_users"
  ("_idx", "email", "id", "name", "_source")
  VALUES ('1790448836467', 'marta@example.com', 'u3', 'Marta O''Neil', TO_CLOB('{"age":45,"tp_doc":"NIT"}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name",
'email' VALUE "email"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_users" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_roles"
  ("_idx", "id", "name")
  VALUES ('1790448836472', 'r1', 'admin')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_roles" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_roles"
  ("_idx", "id", "name")
  VALUES ('1790448836527', 'r2', 'editor')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_roles" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_orders"
  ("_idx", "id", "user_id", "_source")
  VALUES ('1790448836530', 'o1', 'u1', TO_CLOB('{"amount":100.5}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'user_id' VALUE "user_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_orders" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_orders"
  ("_idx", "id", "user_id", "_source")
  VALUES ('1790448836587', 'o2', 'u1', TO_CLOB('{"amount":200}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'user_id' VALUE "user_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_orders" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_orders"
  ("_idx", "id", "user_id", "_source")
  VALUES ('1790448836594', 'o3', 'u3', TO_CLOB('{"amount":50}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'user_id' VALUE "user_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_orders" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_orders_items"
  ("id", "order_id", "product")
  VALUES ('i1', 'o1', 'router')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'id' VALUE "id",
'product' VALUE "product"
RETURNING CLOB) AS "result" FROM JSQL."f_orders_items" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_orders_items"
  ("id", "order_id", "product")
  VALUES ('i2', 'o1', 'cable')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'id' VALUE "id",
'product' VALUE "product"
RETURNING CLOB) AS "result" FROM JSQL."f_orders_items" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name", "_source")
  VALUES ('internet', 'p1', 'Plan 200', TO_CLOB('{"price":90000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name", "_source")
  VALUES ('internet', 'p2', 'Plan 500', TO_CLOB('{"price":150000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name", "_source")
  VALUES ('tv', 'p3', 'Decoder', TO_CLOB('{"price":20000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_users_f_roles"
  ("_idx", "role_id", "user_id")
  VALUES ('1790448836663', 'r1', 'u1')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'user_id' VALUE "user_id",
'role_id' VALUE "role_id"
RETURNING CLOB) AS "result" FROM JSQL."f_users_f_roles" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_users_f_roles"
  ("_idx", "role_id", "user_id")
  VALUES ('1790448836671', 'r2', 'u1')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'user_id' VALUE "user_id",
'role_id' VALUE "role_id"
RETURNING CLOB) AS "result" FROM JSQL."f_users_f_roles" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_users_f_roles"
  ("_idx", "role_id", "user_id")
  VALUES ('1790448836674', 'r2', 'u2')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'user_id' VALUE "user_id",
'role_id' VALUE "role_id"
RETURNING CLOB) AS "result" FROM JSQL."f_users_f_roles" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 3. Relaciones y campos calculados · Detail en el select (DefineDetail) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id"
RETURNING CLOB) AS "result"
FROM JSQL."f_orders" A
WHERE A."id" = 'o1'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'product' VALUE A."product"
RETURNING CLOB) AS "result"
FROM JSQL."f_orders_items" A
WHERE A."order_id" = 'o1'
FETCH NEXT 30 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Model.Detail + Query.Detail [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'user_id' VALUE A."user_id"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_orders" A
WHERE A."id" = 'o1'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'product' VALUE A."product"
RETURNING CLOB) AS "result"
FROM JSQL."f_orders_items" A
WHERE A."order_id" = 'o1'
FETCH NEXT 30 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Master en el select (DefineMaster) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id"
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u1'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'name' VALUE A."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_roles" A
INNER JOIN JSQL."f_users_f_roles" B
  ON B."role_id" = A."id"
WHERE B."user_id" = 'u1'
FETCH NEXT 1000 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Master 1 a 1 (rows = 1) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id"
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u2'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'name' VALUE A."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_roles" A
INNER JOIN JSQL."f_users_f_roles" B
  ON B."role_id" = A."id"
WHERE B."user_id" = 'u2'
FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · Model.Master + Model.Bridge [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_roles" A
INNER JOIN JSQL."f_users_f_roles" B
  ON A."id" = B."role_id"
WHERE B."user_id" = 'u2'
FETCH NEXT 1000 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineRollup row (tp_doc → title) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'tp_doc' VALUE SUBSTR(JSON_QUERY(A."_source", '$."tp_doc"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(A."_source", '$."tp_doc"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'title' VALUE A."title"
RETURNING CLOB) AS "result"
FROM JSQL."f_doc_types" A
WHERE A."id" = 'CC'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'title' VALUE A."title"
RETURNING CLOB) AS "result"
FROM JSQL."f_doc_types" A
WHERE A."id" = NULL
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'title' VALUE A."title"
RETURNING CLOB) AS "result"
FROM JSQL."f_doc_types" A
WHERE A."id" = 'NIT'
FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineRollup (count, sum, object) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id"
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u3'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT COUNT(*) AS "count"
FROM JSQL."f_orders" A
WHERE A."user_id" = 'u3'

-- QUERY
SELECT
JSON_OBJECT(
'amount' VALUE SUM(JSON_VALUE(A."_source", '$."amount"' RETURNING NUMBER))
RETURNING CLOB) AS "result"
FROM JSQL."f_orders" A
WHERE A."user_id" = 'u3'
FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'amount' VALUE SUBSTR(JSON_QUERY(A."_source", '$."amount"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(A."_source", '$."amount"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON
RETURNING CLOB) AS "result"
FROM JSQL."f_orders" A
WHERE A."user_id" = 'u3'
FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineCalcFunc + Model.Calc [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u1'
FETCH NEXT 1 ROWS ONLY

-- ========== 3. Relaciones y campos calculados · DefineCalc (script JS) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u3'
FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Select + Where + OrderBy + All [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'age' VALUE SUBSTR(JSON_QUERY(A."_source", '$."age"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(A."_source", '$."age"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) > 18
ORDER BY JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) DESC
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · One / First / Count / Exists [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u2'
FETCH NEXT 1 ROWS ONLY

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
ORDER BY A."id" ASC
FETCH NEXT 2 ROWS ONLY

-- QUERY
SELECT COUNT(*) AS "count"
FROM JSQL."f_users" A

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_users" A
WHERE A."id" = 'u9') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- ========== 4. Consultas · Limit (paginación) / Page [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
ORDER BY A."id" ASC
OFFSET 1 ROWS
FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Hidden (campo oculto en la consulta y en el modelo) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"email":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u1'
FETCH NEXT 1 ROWS ONLY

-- ========== 4. Consultas · Join (fluido) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'user' VALUE U."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_orders" A
INNER JOIN JSQL."f_users" U
  ON A."user_id" = U."id"
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · LeftJoin + GroupBy [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'n' VALUE COUNT(O."id")
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
LEFT JOIN JSQL."f_orders" O
  ON O."user_id" = A."id"
GROUP BY A."id"
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · GroupBy + Having (fluido) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'user_id' VALUE A."user_id",
'n' VALUE COUNT(A."id"),
'total' VALUE SUM(JSON_VALUE(A."_source", '$."amount"' RETURNING NUMBER))
RETURNING CLOB) AS "result"
FROM JSQL."f_orders" A
GROUP BY A."user_id"
HAVING COUNT(A."id") > 1
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query (JSON) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) >= 18
ORDER BY A."name" DESC
FETCH NEXT 10 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from (otro modelo y alias) [pass]

-- QUERY
SELECT
JSON_OBJECT(
'id' VALUE U."id",
'name' VALUE U."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_users" U
WHERE JSON_VALUE(U."_source", '$."age"' RETURNING NUMBER) > 18
ORDER BY U."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from sin esquema [pass]

-- QUERY
SELECT
JSON_OBJECT(
'name' VALUE R."name"
RETURNING CLOB) AS "result"
FROM JSQL."f_roles" R
ORDER BY R."name" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Model.Query con from + join + groups [pass]

-- QUERY
SELECT
JSON_OBJECT(
'name' VALUE U."name",
'n' VALUE COUNT(O."id")
RETURNING CLOB) AS "result"
FROM JSQL."f_users" U
INNER JOIN JSQL."f_orders" O
  ON O."user_id" = U."id"
GROUP BY U."name"
ORDER BY U."name" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 4. Consultas · Test (genera el SQL sin ejecutarlo) / Debug [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" = 'u1'
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Eq [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."name" = 'Ana'
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Neg [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."name" != 'Ana'
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Less [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) < 30
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · LessEq [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) <= 30
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · More [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) > 30
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · MoreEq [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) >= 30
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Like [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE UPPER(A."name") LIKE UPPER('%an%')
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · In [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" IN ('u1', 'u3')
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · NotIn [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."id" NOT IN ('u1', 'u3')
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Is (NULL) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."nickname"') IS NULL
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · IsNot (NULL) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) IS NOT NULL
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Null [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."nickname"') IS NULL
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · NotNull [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."email" IS NOT NULL
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Between [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) BETWEEN 18 AND 40
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · NotBetween [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) NOT BETWEEN 18 AND 40
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · Where (genérico) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."name" = 'Luis'
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- ========== 5. Condiciones · And / Or (conectores) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE A."name" = 'Ana'
  OR A."name" = 'Luis'
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null,"password":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name",
'email' VALUE A."email"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_users" A
WHERE JSON_VALUE(A."_source", '$."age"' RETURNING NUMBER) > 18
  AND UPPER(A."name") LIKE UPPER('%neil%')
FETCH NEXT 1000 ROWS ONLY

-- ========== 6. Comandos · Insert (RETURNING) [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name", "_source")
  VALUES ('equipos', 'p4', 'Router Wi-Fi 6', TO_CLOB('{"price":350000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 6. Comandos · Bulk [pass]

-- BULK
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name", "_source")
  VALUES ('tv', 'p5', 'Cable HDMI', TO_CLOB('{"price":15000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- BULK
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name", "_source")
  VALUES ('tv', 'p6', 'Control', TO_CLOB('{"price":10000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 6. Comandos · Update + Where [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 'p1'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_products"
  SET "category" = 'internet',
    "name" = 'Plan 200',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"price":99000,"promo":true}') RETURNING CLOB)
  WHERE "id" = 'p1'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 6. Comandos · Update + Where + Or (varias filas) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 'p5'
  OR A."id" = 'p6'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_products"
  SET "category" = 'tv',
    "name" = 'Cable HDMI',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"price":15000,"stock":5}') RETURNING CLOB)
  WHERE "id" = 'p5'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_products"
  SET "category" = 'tv',
    "name" = 'Control',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"price":10000,"stock":5}') RETURNING CLOB)
  WHERE "id" = 'p6'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 6. Comandos · Upsert (inserta y luego actualiza) [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_products" A
WHERE A."id" = 'p7') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("category", "id", "name")
  VALUES ('tv', 'p7', 'Antena')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_products" A
WHERE A."id" = 'p7') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 'p7'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_products"
  SET "category" = 'tv',
    "name" = 'Antena HD'
  WHERE "id" = 'p7'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 6. Comandos · Return (campos del RETURNING) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 'p4'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_products"
  SET "category" = 'equipos',
    "name" = 'Router Wi-Fi 6',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"price":350000,"stock":7}') RETURNING CLOB)
  WHERE "id" = 'p4'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'id' VALUE "id"
RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 6. Comandos · Delete (devuelve la fila borrada) [pass]

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 'p6'
FETCH NEXT 1000 ROWS ONLY

-- DELETE
DECLARE
  c SYS_REFCURSOR;
BEGIN
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE "id" = 'p6';
  DELETE FROM JSQL."f_products" WHERE "id" = 'p6';
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_products" A
WHERE A."id" = 'p6') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- ========== 6. Comandos · Test (no ejecuta) + ToJson [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("id", "name")
  VALUES ('p8', 'No se guarda')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_products" A
WHERE A."id" = 'p8') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- ========== 7. Triggers · Before/After Insert, Update, Delete e InsertOrUpdate [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."f_events" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."f_events" ADD CONSTRAINT "jsql_f_events_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_events_status_idx" ON JSQL."f_events" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_f_events__idx_idx" ON JSQL."f_events" ("_idx")';
END;

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_events"
  ("_idx", "id", "name", "_source")
  VALUES ('1790448836898', 'e1', 'alta', TO_CLOB('{"stage":"before_insert","touched":true}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_events" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_events" A
WHERE A."id" = 'e1'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_events"
  SET "name" = 'cambio',
    "status" = 'active',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"stage":"before_update","touched":true}') RETURNING CLOB)
  WHERE "id" = 'e1'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_events" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_events" A
WHERE A."id" = 'e1'
FETCH NEXT 1000 ROWS ONLY

-- DELETE
DECLARE
  c SYS_REFCURSOR;
BEGIN
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_events" WHERE "id" = 'e1';
  DELETE FROM JSQL."f_events" WHERE "id" = 'e1';
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 7. Triggers · Trigger del comando + error que aborta [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_events"
  ("_idx", "id", "name", "_source")
  VALUES ('1790448836939', 'e2', 'cmd', TO_CLOB('{"source":"command","stage":"before_insert","touched":true}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_events" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_events" A
WHERE A."id" = 'e3') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- ========== 7. Triggers · Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_events"
  ("_idx", "id", "name", "_source")
  VALUES ('1790448836945', 'e4', 'js', TO_CLOB('{"js":"before_insert","stage":"before_insert","touched":true}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_events" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'name' VALUE A."name"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_events" A
WHERE A."id" = 'e4'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_events"
  SET "name" = 'js2',
    "status" = 'active',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"js":"before_update","stage":"before_update","touched":true}') RETURNING CLOB)
  WHERE "id" = 'e4'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_events" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 8. Transacciones · NewTx + ExecTx + Rollback [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("id", "name")
  VALUES ('t1', 'rollback')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."f_products" A
WHERE A."id" = 't1') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- ========== 8. Transacciones · NewTx + ExecTx + Commit [pass]

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."f_products"
  ("id", "name")
  VALUES ('t2', 'commit')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 't2'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."f_products"
  SET "name" = 'commit 2'
  WHERE "id" = 't2'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE "id",
'name' VALUE "name",
'category' VALUE "category"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."f_products" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), JSON_OBJECT(
'id' VALUE A."id",
'name' VALUE A."name",
'category' VALUE A."category"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."f_products" A
WHERE A."id" = 't2'
FETCH NEXT 1 ROWS ONLY

-- ========== 9. Series · DefineSeries [pass]

-- DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."series" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "tag" VARCHAR2(80) DEFAULT NULL,
  "format" VARCHAR2(255) DEFAULT NULL,
  "value" NUMBER(19) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."series" ADD CONSTRAINT "jsql_series_pkey" PRIMARY KEY ("tag")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_series__idx_idx" ON JSQL."series" ("_idx")';
END;

-- ========== 9. Series · SetSeries + GetSeries [pass]

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."series" A
WHERE A."tag" = 'invoice') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."series"
  ("_idx", "created_at", "format", "tag", "updated_at", "value")
  VALUES ('1790448836980', TO_TIMESTAMP('2026-09-26 13:53:56.980150', 'YYYY-MM-DD HH24:MI:SS.FF'), 'FAC-%05d', 'invoice', TO_TIMESTAMP('2026-09-26 13:53:56.980150', 'YYYY-MM-DD HH24:MI:SS.FF'), 10)
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'tag' VALUE "tag",
'format' VALUE "format",
'value' VALUE "value"
RETURNING CLOB) AS "result" FROM JSQL."series" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'tag' VALUE A."tag",
'format' VALUE A."format",
'value' VALUE A."value"
RETURNING CLOB) AS "result"
FROM JSQL."series" A
WHERE A."tag" = 'invoice'
FETCH NEXT 1 ROWS ONLY

-- ========== 9. Series · GenValue + GenSerie [pass]

-- QUERY
SELECT
JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'tag' VALUE A."tag",
'format' VALUE A."format",
'value' VALUE A."value"
RETURNING CLOB) AS "result"
FROM JSQL."series" A
WHERE A."tag" = 'invoice'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."series"
  SET "created_at" = TO_TIMESTAMP('2026-09-26 13:53:56.980150', 'YYYY-MM-DD HH24:MI:SS.FF'),
    "format" = 'FAC-%05d',
    "updated_at" = TO_TIMESTAMP('2026-09-26 13:53:56.990703', 'YYYY-MM-DD HH24:MI:SS.FF'),
    "value" = 11
  WHERE "tag" = 'invoice'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'tag' VALUE "tag",
'format' VALUE "format",
'value' VALUE "value"
RETURNING CLOB) AS "result" FROM JSQL."series" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT CASE WHEN EXISTS(SELECT 1
FROM JSQL."series" A
WHERE A."tag" = 'invoice') THEN 'true' ELSE 'false' END AS "exists" FROM DUAL

-- QUERY
SELECT
JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'tag' VALUE A."tag",
'format' VALUE A."format",
'value' VALUE A."value"
RETURNING CLOB) AS "result"
FROM JSQL."series" A
WHERE A."tag" = 'invoice'
FETCH NEXT 1000 ROWS ONLY

-- UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."series"
  SET "created_at" = TO_TIMESTAMP('2026-09-26 13:53:56.980150', 'YYYY-MM-DD HH24:MI:SS.FF'),
    "format" = 'FAC-%05d',
    "updated_at" = TO_TIMESTAMP('2026-09-26 13:53:56.997061', 'YYYY-MM-DD HH24:MI:SS.FF'),
    "value" = 12
  WHERE "tag" = 'invoice'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'tag' VALUE "tag",
'format' VALUE "format",
'value' VALUE "value"
RETURNING CLOB) AS "result" FROM JSQL."series" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- ========== 9. Series · DeleteSeries [pass]

-- QUERY
SELECT
JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'tag' VALUE A."tag",
'format' VALUE A."format",
'value' VALUE A."value"
RETURNING CLOB) AS "result"
FROM JSQL."series" A
WHERE A."tag" = 'invoice'
FETCH NEXT 1000 ROWS ONLY

-- DELETE
DECLARE
  c SYS_REFCURSOR;
BEGIN
  OPEN c FOR SELECT JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'tag' VALUE "tag",
'format' VALUE "format",
'value' VALUE "value"
RETURNING CLOB) AS "result" FROM JSQL."series" WHERE "tag" = 'invoice';
  DELETE FROM JSQL."series" WHERE "tag" = 'invoice';
  DBMS_SQL.RETURN_RESULT(c);
END;

-- QUERY
SELECT
JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'tag' VALUE A."tag",
'format' VALUE A."format",
'value' VALUE A."value"
RETURNING CLOB) AS "result"
FROM JSQL."series" A
WHERE A."tag" = 'invoice'
FETCH NEXT 1 ROWS ONLY


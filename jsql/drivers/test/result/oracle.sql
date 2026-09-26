-- TestInsertUpdate · DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."transfers" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "kind" VARCHAR2(80) DEFAULT NULL,
  "code" VARCHAR2(80) DEFAULT NULL,
  "client_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."transfers" ADD CONSTRAINT "jsql_transfers_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_transfers_status_idx" ON JSQL."transfers" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_transfers__idx_idx" ON JSQL."transfers" ("_idx")';
END;

-- TestInsertUpdate · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."transfers"
  ("_idx", "client_id", "code", "id", "kind", "status", "_source")
  VALUES ('1790443471105', '1000000001', '00000009', '1e0fffc3-448a-47b8-92e1-97c572ba388c', 'transfers', 'en_process', TO_CLOB('{"app_id":"Vista360","appointmentDate":"2026-07-24","billingAddress":null,"caption":"Traslate_Order 00000009","channel":"","createdAt":"2026-07-23T20:06:44.670Z","data":{"appointment_date":"2026-07-24","create_automatic_ticket":true,"extended_attribute_values":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extended_attributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"P') || TO_CLOB('UNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-') || TO_CLOB('zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_d') || TO_CLOB('ocumento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERA') || TO_CLOB('DOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE D') || TO_CLOB('OCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":') || TO_CLOB('"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"freshdeskTicketId":100001,"plans_result":null,"selected_account":"10000000001","selected_plan":') || TO_CLOB('null,"selected_service":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"stratum_change_attachments":[],"transfer_address":{"address":"CR 2 CL 3-04 APT1","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"department_name":"VALLE DEL CAUCA","depto_id":"76","formatted_address":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","municipality_id":"76520","municipality_name":"PALMIRA","neighborhood_id":3009,"neighborhood_name":"RECREO","population_center":"","property_type_id":1,"property_type_name":"RESIDENCIAL","stratum":"4"},"transfer_address_form":{"department":{"attribute":"department","id":"b9cc5d75-5839-402a-bb2d-dbddbccd8948","idCode":"76","i') || TO_CLOB('dTable":"24","label":"VALLE DEL CAUCA","name":"VALLE DEL CAUCA","value":"VALLE DEL CAUCA"},"municipality":{"attribute":"municipality","id":"7d58f5a1-964e-42c1-bf70-55e20244957b","idCode":"76520","idTable":"1032","label":"PALMIRA","name":"PALMIRA","value":"PALMIRA"},"neighborhood":{"attribute":"neighborhood","id":3009,"label":"RECREO","name":"RECREO","value":"RECREO"},"numberPrimary":"2","numberSecondary":"3","observations":"","plate":"04 APT1","propertyType":{"id":1,"name":"RESIDENCIAL"},"routePrimary":{"id":"CR","label":"CARRERA","name":"CR"},"routeSecondary":{"id":"CL","label":"CALLE","name":"CL"},"stratum":{"id":"4","name":"4"}},"transfer_address_map_selection":{"accuracy":"approximate","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"formattedAddress":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","source":"address"},"transfer_address_preview":"CR 2 CL 3-04 APT1","transfer_address_validation":{"cycle":"","message":"Reserva obtenida por dirección.","reserved_port_id":"') || TO_CLOB('100002","selected_stratum":"4","suggested_stratum":"-","typeOfService":"SERVICIO FISICO"},"transfer_resumed_from_history":true,"type_of_service_sugerido":"SERVICIO FISICO"},"description":"Traslate_Order 00000009","endDate":"","extendedAttributeValues":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extendedAttributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VE') || TO_CLOB('NTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$') || TO_CLOB('%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":') || TO_CLOB('{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL') || TO_CLOB('","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"') || TO_CLOB('},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEG') || TO_CLOB('ORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"maxStepReached":3,"orderCode":"00000009","payload":null,"plansResult":null,"project_id":"-1","selectedAcc') || TO_CLOB('ount":"10000000001","selectedAvailabilityDate":"","selectedAvailabilitySlot":null,"selectedPlan":null,"selectedService":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"serviceId":"1e0fffc3-448a-47b8-92e1-97c572ba388c","step":3,"transferAddress":null}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'kind' VALUE "kind",
'code' VALUE "code",
'client_id' VALUE "client_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."transfers" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestInsertUpdate · SQL
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'kind' VALUE A."kind",
'code' VALUE A."code",
'client_id' VALUE A."client_id"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."transfers" A
WHERE A."id" = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
FETCH NEXT 1 ROWS ONLY

-- TestInsertUpdate · SQL
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'code' VALUE A."code",
'municipality' VALUE SUBSTR(JSON_QUERY(A."_source", '$."data"."transfer_address"."municipality_name"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(A."_source", '$."data"."transfer_address"."municipality_name"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON,
'plan' VALUE SUBSTR(JSON_QUERY(A."_source", '$."extendedAttributeValues"."plan_comercial"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(A."_source", '$."extendedAttributeValues"."plan_comercial"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON,
'step' VALUE SUBSTR(JSON_QUERY(A."_source", '$."step"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(A."_source", '$."step"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON
RETURNING CLOB) AS "result"
FROM JSQL."transfers" A
WHERE A."kind" = 'transfers'
  AND JSON_VALUE(A."_source", '$."data"."transfer_address"."stratum"') = '4'
  AND JSON_VALUE(A."_source", '$."step"' RETURNING NUMBER) = 3
FETCH NEXT 1000 ROWS ONLY

-- TestInsertUpdate · SQL
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'kind' VALUE A."kind",
'code' VALUE A."code",
'client_id' VALUE A."client_id"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."transfers" A
WHERE A."id" = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
FETCH NEXT 1000 ROWS ONLY

-- TestInsertUpdate · UPDATE
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  UPDATE JSQL."transfers"
  SET "client_id" = '1000000001',
    "code" = '00000010',
    "kind" = 'transfers',
    "status" = 'done',
    "_source" = JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"app_id":"Vista360","appointmentDate":"2026-07-24","billingAddress":null,"caption":"Traslado \"urgente\" de O''Brien","channel":"","createdAt":"2026-07-23T20:06:44.670Z","data":{"selected_service":{"status":"inactivo"},"transfer_address":{"notes":"C:\\ruta\\nueva","stratum":"5"}},"description":"Traslate_Order 00000009","endDate":"","extendedAttributeValues":{"tipo_cliente":"INQUILINO"},"extendedAttributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \') || TO_CLOB('"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"c') || TO_CLOB('liente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"10000') || TO_CLOB('00001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITI') || TO_CLOB('ONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"') || TO_CLOB('ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR') || TO_CLOB('","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"maxStepReached":3,"orderCode":"00000009","payload":null,"plansResult":null,"project_id":"-1","selectedAccount":"10000000001","selectedAvailabilityDate":"","selectedAvailabilitySlot":null,"selectedPlan":null,"selectedService":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":') || TO_CLOB('"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"serviceId":"1e0fffc3-448a-47b8-92e1-97c572ba388c","step":4,"transferAddress":null}') RETURNING CLOB)
  WHERE "id" = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'kind' VALUE "kind",
'code' VALUE "code",
'client_id' VALUE "client_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."transfers" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestInsertUpdate · SQL
SELECT
JSON_MERGEPATCH(JSON_MERGEPATCH(NVL(A."_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE A."created_at",
'updated_at' VALUE A."updated_at",
'status' VALUE A."status",
'id' VALUE A."id",
'kind' VALUE A."kind",
'code' VALUE A."code",
'client_id' VALUE A."client_id"
RETURNING CLOB) RETURNING CLOB) AS "result"
FROM JSQL."transfers" A
WHERE A."id" = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
FETCH NEXT 1 ROWS ONLY

-- TestJoinGroupHaving · DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."clients" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."clients" ADD CONSTRAINT "jsql_clients_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_clients_status_idx" ON JSQL."clients" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_clients__idx_idx" ON JSQL."clients" ("_idx")';
END;

-- TestJoinGroupHaving · DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."plans" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "name" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."plans" ADD CONSTRAINT "jsql_plans_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_plans_status_idx" ON JSQL."plans" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_plans__idx_idx" ON JSQL."plans" ("_idx")';
END;

-- TestJoinGroupHaving · DDL
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE JSQL."subscriptions" (
  "created_at" TIMESTAMP DEFAULT NULL,
  "updated_at" TIMESTAMP DEFAULT NULL,
  "status" VARCHAR2(255) DEFAULT ''active'',
  "id" VARCHAR2(80) DEFAULT NULL,
  "_source" CLOB DEFAULT TO_CLOB(''{}'') CHECK ("_source" IS JSON),
  "client_id" VARCHAR2(80) DEFAULT NULL,
  "plan_id" VARCHAR2(80) DEFAULT NULL,
  "_idx" VARCHAR2(80) DEFAULT NULL
)';
  EXECUTE IMMEDIATE 'ALTER TABLE JSQL."subscriptions" ADD CONSTRAINT "jsql_subscriptions_pkey" PRIMARY KEY ("id")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_subscriptions_status_idx" ON JSQL."subscriptions" ("status")';
  EXECUTE IMMEDIATE 'CREATE INDEX "jsql_subscriptions__idx_idx" ON JSQL."subscriptions" ("_idx")';
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."clients"
  ("_idx", "id", "name", "_source")
  VALUES ('1790443471598', 'c1', 'Ana', TO_CLOB('{"city":"PALMIRA"}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."clients" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."clients"
  ("_idx", "id", "name", "_source")
  VALUES ('1790443471625', 'c2', 'Luis', TO_CLOB('{"city":"CALI"}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."clients" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."clients"
  ("_idx", "id", "name", "_source")
  VALUES ('1790443471629', 'c3', 'Marta O''Neil', TO_CLOB('{"city":"PALMIRA"}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."clients" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."plans"
  ("_idx", "id", "name", "_source")
  VALUES ('1790443471633', 'p1', 'PLAN 200', TO_CLOB('{"price":90000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."plans" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."plans"
  ("_idx", "id", "name", "_source")
  VALUES ('1790443471650', 'p2', 'PLAN 500', TO_CLOB('{"price":150000}'))
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'name' VALUE "name"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."plans" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."subscriptions"
  ("_idx", "client_id", "id", "plan_id")
  VALUES ('1790443471654', 'c1', 's1', 'p1')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'client_id' VALUE "client_id",
'plan_id' VALUE "plan_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."subscriptions" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."subscriptions"
  ("_idx", "client_id", "id", "plan_id")
  VALUES ('1790443471672', 'c1', 's2', 'p2')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'client_id' VALUE "client_id",
'plan_id' VALUE "plan_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."subscriptions" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."subscriptions"
  ("_idx", "client_id", "id", "plan_id")
  VALUES ('1790443471677', 'c2', 's3', 'p1')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'client_id' VALUE "client_id",
'plan_id' VALUE "plan_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."subscriptions" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · INSERT
DECLARE
  rs SYS.ODCIVARCHAR2LIST;
  c SYS_REFCURSOR;
BEGIN
  INSERT INTO JSQL."subscriptions"
  ("_idx", "client_id", "id", "plan_id")
  VALUES ('1790443471680', 'c3', 's4', 'p2')
  RETURNING ROWIDTOCHAR(ROWID) BULK COLLECT INTO rs;
  OPEN c FOR SELECT JSON_MERGEPATCH(JSON_MERGEPATCH(NVL("_source", TO_CLOB('{}')), TO_CLOB('{"_idx":null}') RETURNING CLOB), JSON_OBJECT(
'created_at' VALUE "created_at",
'updated_at' VALUE "updated_at",
'status' VALUE "status",
'id' VALUE "id",
'client_id' VALUE "client_id",
'plan_id' VALUE "plan_id"
RETURNING CLOB) RETURNING CLOB) AS "result" FROM JSQL."subscriptions" WHERE ROWIDTOCHAR(ROWID) IN (SELECT COLUMN_VALUE FROM TABLE(rs));
  DBMS_SQL.RETURN_RESULT(c);
END;

-- TestJoinGroupHaving · SQL
SELECT
JSON_OBJECT(
'id' VALUE A."id",
'client' VALUE C."name",
'city' VALUE SUBSTR(JSON_QUERY(C."_source", '$."city"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(C."_source", '$."city"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON,
'plan' VALUE P."name",
'price' VALUE SUBSTR(JSON_QUERY(P."_source", '$."price"' RETURNING CLOB WITH ARRAY WRAPPER), 2, LENGTH(JSON_QUERY(P."_source", '$."price"' RETURNING CLOB WITH ARRAY WRAPPER)) - 2) FORMAT JSON
RETURNING CLOB) AS "result"
FROM JSQL."subscriptions" A
INNER JOIN JSQL."clients" C
  ON A."client_id" = C."id"
INNER JOIN JSQL."plans" P
  ON A."plan_id" = P."id"
ORDER BY A."id" ASC
FETCH NEXT 1000 ROWS ONLY

-- TestJoinGroupHaving · SQL
SELECT
JSON_OBJECT(
'city' VALUE JSON_VALUE(C."_source", '$."city"'),
'total' VALUE COUNT(A."id"),
'amount' VALUE SUM(JSON_VALUE(P."_source", '$."price"' RETURNING NUMBER))
RETURNING CLOB) AS "result"
FROM JSQL."subscriptions" A
INNER JOIN JSQL."clients" C
  ON A."client_id" = C."id"
INNER JOIN JSQL."plans" P
  ON A."plan_id" = P."id"
GROUP BY JSON_VALUE(C."_source", '$."city"')
ORDER BY JSON_VALUE(C."_source", '$."city"') ASC
FETCH NEXT 1000 ROWS ONLY

-- TestJoinGroupHaving · SQL
SELECT
JSON_OBJECT(
'city' VALUE JSON_VALUE(C."_source", '$."city"'),
'total' VALUE COUNT(A."id"),
'amount' VALUE SUM(JSON_VALUE(P."_source", '$."price"' RETURNING NUMBER))
RETURNING CLOB) AS "result"
FROM JSQL."subscriptions" A
INNER JOIN JSQL."clients" C
  ON A."client_id" = C."id"
INNER JOIN JSQL."plans" P
  ON A."plan_id" = P."id"
GROUP BY JSON_VALUE(C."_source", '$."city"')
HAVING COUNT(A."id") > 1
ORDER BY JSON_VALUE(C."_source", '$."city"') ASC
FETCH NEXT 1000 ROWS ONLY

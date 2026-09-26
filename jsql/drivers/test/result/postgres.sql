-- TestInsertUpdate · DDL
CREATE SCHEMA IF NOT EXISTS jsql_test;

CREATE TABLE IF NOT EXISTS jsql_test.transfers (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  kind VARCHAR(80) DEFAULT NULL,
  code VARCHAR(80) DEFAULT NULL,
  client_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_test.transfers ADD CONSTRAINT jsql_test_transfers_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_test_transfers_status_idx ON jsql_test.transfers USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_test_transfers__idx_idx ON jsql_test.transfers USING BTREE (_idx);

-- TestInsertUpdate · INSERT
INSERT INTO jsql_test.transfers
  (_idx, client_id, code, id, kind, status, _source)
VALUES
  ('1790443470816', '1000000001', '00000009', '1e0fffc3-448a-47b8-92e1-97c572ba388c', 'transfers', 'en_process', '{"app_id":"Vista360","appointmentDate":"2026-07-24","billingAddress":null,"caption":"Traslate_Order 00000009","channel":"","createdAt":"2026-07-23T20:06:44.670Z","data":{"appointment_date":"2026-07-24","create_automatic_ticket":true,"extended_attribute_values":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extended_attributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"freshdeskTicketId":100001,"plans_result":null,"selected_account":"10000000001","selected_plan":null,"selected_service":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"stratum_change_attachments":[],"transfer_address":{"address":"CR 2 CL 3-04 APT1","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"department_name":"VALLE DEL CAUCA","depto_id":"76","formatted_address":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","municipality_id":"76520","municipality_name":"PALMIRA","neighborhood_id":3009,"neighborhood_name":"RECREO","population_center":"","property_type_id":1,"property_type_name":"RESIDENCIAL","stratum":"4"},"transfer_address_form":{"department":{"attribute":"department","id":"b9cc5d75-5839-402a-bb2d-dbddbccd8948","idCode":"76","idTable":"24","label":"VALLE DEL CAUCA","name":"VALLE DEL CAUCA","value":"VALLE DEL CAUCA"},"municipality":{"attribute":"municipality","id":"7d58f5a1-964e-42c1-bf70-55e20244957b","idCode":"76520","idTable":"1032","label":"PALMIRA","name":"PALMIRA","value":"PALMIRA"},"neighborhood":{"attribute":"neighborhood","id":3009,"label":"RECREO","name":"RECREO","value":"RECREO"},"numberPrimary":"2","numberSecondary":"3","observations":"","plate":"04 APT1","propertyType":{"id":1,"name":"RESIDENCIAL"},"routePrimary":{"id":"CR","label":"CARRERA","name":"CR"},"routeSecondary":{"id":"CL","label":"CALLE","name":"CL"},"stratum":{"id":"4","name":"4"}},"transfer_address_map_selection":{"accuracy":"approximate","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"formattedAddress":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","source":"address"},"transfer_address_preview":"CR 2 CL 3-04 APT1","transfer_address_validation":{"cycle":"","message":"Reserva obtenida por dirección.","reserved_port_id":"100002","selected_stratum":"4","suggested_stratum":"-","typeOfService":"SERVICIO FISICO"},"transfer_resumed_from_history":true,"type_of_service_sugerido":"SERVICIO FISICO"},"description":"Traslate_Order 00000009","endDate":"","extendedAttributeValues":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extendedAttributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"maxStepReached":3,"orderCode":"00000009","payload":null,"plansResult":null,"project_id":"-1","selectedAccount":"10000000001","selectedAvailabilityDate":"","selectedAvailabilitySlot":null,"selectedPlan":null,"selectedService":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"serviceId":"1e0fffc3-448a-47b8-92e1-97c572ba388c","step":3,"transferAddress":null}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'kind', kind,
'code', code,
'client_id', client_id
) AS result;

-- TestInsertUpdate · SQL
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'kind', A.kind,
'code', A.code,
'client_id', A.client_id
) AS result
FROM jsql_test.transfers AS A
WHERE A.id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
LIMIT 1;

-- TestInsertUpdate · SQL
SELECT
jsonb_build_object(
'id', A.id,
'code', A.code,
'municipality', A._source->'data'->'transfer_address'->'municipality_name',
'plan', A._source->'extendedAttributeValues'->'plan_comercial',
'step', A._source->'step'
) AS result
FROM jsql_test.transfers AS A
WHERE A.kind = 'transfers'
  AND A._source->'data'->'transfer_address'->>'stratum' = '4'
  AND (A._source->>'step')::NUMERIC = 3
LIMIT 1000;

-- TestInsertUpdate · SQL
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'kind', A.kind,
'code', A.code,
'client_id', A.client_id
) AS result
FROM jsql_test.transfers AS A
WHERE A.id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
LIMIT 1000;

-- TestInsertUpdate · UPDATE
UPDATE jsql_test.transfers
SET
  client_id = '1000000001',
  code = '00000010',
  created_at = NULL,
  kind = 'transfers',
  status = 'done',
  updated_at = NULL,
  _source = jsonb_set(jsonb_set(jsonb_set(jsonb_set(COALESCE(_source, '{}'::jsonb) || '{"app_id":"Vista360","appointmentDate":"2026-07-24","billingAddress":null,"caption":"Traslado \"urgente\" de O''Brien","channel":"","createdAt":"2026-07-23T20:06:44.670Z","data":{"appointment_date":"2026-07-24","create_automatic_ticket":true,"extended_attribute_values":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extended_attributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"freshdeskTicketId":100001,"plans_result":null,"selected_account":"10000000001","selected_plan":null,"selected_service":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"stratum_change_attachments":[],"transfer_address":{"address":"CR 2 CL 3-04 APT1","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"department_name":"VALLE DEL CAUCA","depto_id":"76","formatted_address":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","municipality_id":"76520","municipality_name":"PALMIRA","neighborhood_id":3009,"neighborhood_name":"RECREO","population_center":"","property_type_id":1,"property_type_name":"RESIDENCIAL","stratum":"4"},"transfer_address_form":{"department":{"attribute":"department","id":"b9cc5d75-5839-402a-bb2d-dbddbccd8948","idCode":"76","idTable":"24","label":"VALLE DEL CAUCA","name":"VALLE DEL CAUCA","value":"VALLE DEL CAUCA"},"municipality":{"attribute":"municipality","id":"7d58f5a1-964e-42c1-bf70-55e20244957b","idCode":"76520","idTable":"1032","label":"PALMIRA","name":"PALMIRA","value":"PALMIRA"},"neighborhood":{"attribute":"neighborhood","id":3009,"label":"RECREO","name":"RECREO","value":"RECREO"},"numberPrimary":"2","numberSecondary":"3","observations":"","plate":"04 APT1","propertyType":{"id":1,"name":"RESIDENCIAL"},"routePrimary":{"id":"CR","label":"CARRERA","name":"CR"},"routeSecondary":{"id":"CL","label":"CALLE","name":"CL"},"stratum":{"id":"4","name":"4"}},"transfer_address_map_selection":{"accuracy":"approximate","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"formattedAddress":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","source":"address"},"transfer_address_preview":"CR 2 CL 3-04 APT1","transfer_address_validation":{"cycle":"","message":"Reserva obtenida por dirección.","reserved_port_id":"100002","selected_stratum":"4","suggested_stratum":"-","typeOfService":"SERVICIO FISICO"},"transfer_resumed_from_history":true,"type_of_service_sugerido":"SERVICIO FISICO"},"description":"Traslate_Order 00000009","endDate":"","extendedAttributeValues":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extendedAttributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"maxStepReached":3,"orderCode":"00000009","payload":null,"plansResult":null,"project_id":"-1","selectedAccount":"10000000001","selectedAvailabilityDate":"","selectedAvailabilitySlot":null,"selectedPlan":null,"selectedService":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"serviceId":"1e0fffc3-448a-47b8-92e1-97c572ba388c","step":4,"transferAddress":null}'::jsonb, '{"data","selected_service","status"}', '"inactivo"'::jsonb, true), '{"data","transfer_address","notes"}', '"C:\\ruta\\nueva"'::jsonb, true), '{"data","transfer_address","stratum"}', '"5"'::jsonb, true), '{"extendedAttributeValues","tipo_cliente"}', '"INQUILINO"'::jsonb, true)
WHERE id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'kind', kind,
'code', code,
'client_id', client_id
) AS result;

-- TestInsertUpdate · SQL
SELECT
(A._source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', A.created_at,
'updated_at', A.updated_at,
'status', A.status,
'id', A.id,
'kind', A.kind,
'code', A.code,
'client_id', A.client_id
) AS result
FROM jsql_test.transfers AS A
WHERE A.id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
LIMIT 1;

-- TestJoinGroupHaving · DDL
CREATE SCHEMA IF NOT EXISTS jsql_test;

CREATE TABLE IF NOT EXISTS jsql_test.clients (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_test.clients ADD CONSTRAINT jsql_test_clients_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_test_clients_status_idx ON jsql_test.clients USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_test_clients__idx_idx ON jsql_test.clients USING BTREE (_idx);

-- TestJoinGroupHaving · DDL
CREATE SCHEMA IF NOT EXISTS jsql_test;

CREATE TABLE IF NOT EXISTS jsql_test.plans (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  name VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_test.plans ADD CONSTRAINT jsql_test_plans_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_test_plans_status_idx ON jsql_test.plans USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_test_plans__idx_idx ON jsql_test.plans USING BTREE (_idx);

-- TestJoinGroupHaving · DDL
CREATE SCHEMA IF NOT EXISTS jsql_test;

CREATE TABLE IF NOT EXISTS jsql_test.subscriptions (
  created_at TIMESTAMP DEFAULT NULL,
  updated_at TIMESTAMP DEFAULT NULL,
  status VARCHAR(255) DEFAULT 'active',
  id VARCHAR(80) DEFAULT NULL,
  _source JSONB DEFAULT '{}',
  client_id VARCHAR(80) DEFAULT NULL,
  plan_id VARCHAR(80) DEFAULT NULL,
  _idx VARCHAR(80) DEFAULT NULL
);

ALTER TABLE jsql_test.subscriptions ADD CONSTRAINT jsql_test_subscriptions_pkey PRIMARY KEY (id);
CREATE INDEX IF NOT EXISTS jsql_test_subscriptions_status_idx ON jsql_test.subscriptions USING BTREE (status);
CREATE INDEX IF NOT EXISTS jsql_test_subscriptions__idx_idx ON jsql_test.subscriptions USING BTREE (_idx);

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.clients
  (_idx, id, name, _source)
VALUES
  ('1790443471464', 'c1', 'Ana', '{"city":"PALMIRA"}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.clients
  (_idx, id, name, _source)
VALUES
  ('1790443471468', 'c2', 'Luis', '{"city":"CALI"}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.clients
  (_idx, id, name, _source)
VALUES
  ('1790443471469', 'c3', 'Marta O''Neil', '{"city":"PALMIRA"}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.plans
  (_idx, id, name, _source)
VALUES
  ('1790443471470', 'p1', 'PLAN 200', '{"price":90000}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.plans
  (_idx, id, name, _source)
VALUES
  ('1790443471474', 'p2', 'PLAN 500', '{"price":150000}'::jsonb)
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'name', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471475', 'c1', 's1', 'p1')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'client_id', client_id,
'plan_id', plan_id
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471477', 'c1', 's2', 'p2')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'client_id', client_id,
'plan_id', plan_id
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471478', 'c2', 's3', 'p1')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'client_id', client_id,
'plan_id', plan_id
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO jsql_test.subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471479', 'c3', 's4', 'p2')
RETURNING (_source - '{"_idx"}'::text[]) ||
jsonb_build_object(
'created_at', created_at,
'updated_at', updated_at,
'status', status,
'id', id,
'client_id', client_id,
'plan_id', plan_id
) AS result;

-- TestJoinGroupHaving · SQL
SELECT
jsonb_build_object(
'id', A.id,
'client', C.name,
'city', C._source->'city',
'plan', P.name,
'price', P._source->'price'
) AS result
FROM jsql_test.subscriptions AS A
INNER JOIN jsql_test.clients AS C
  ON A.client_id = C.id
INNER JOIN jsql_test.plans AS P
  ON A.plan_id = P.id
ORDER BY A.id ASC
LIMIT 1000;

-- TestJoinGroupHaving · SQL
SELECT
jsonb_build_object(
'city', C._source->>'city',
'total', COUNT(A.id)::bigint,
'amount', COALESCE(SUM((P._source->>'price')::numeric), 0)::double precision
) AS result
FROM jsql_test.subscriptions AS A
INNER JOIN jsql_test.clients AS C
  ON A.client_id = C.id
INNER JOIN jsql_test.plans AS P
  ON A.plan_id = P.id
GROUP BY C._source->>'city'
ORDER BY C._source->>'city' ASC
LIMIT 1000;

-- TestJoinGroupHaving · SQL
SELECT
jsonb_build_object(
'city', C._source->>'city',
'total', COUNT(A.id)::bigint,
'amount', COALESCE(SUM((P._source->>'price')::numeric), 0)::double precision
) AS result
FROM jsql_test.subscriptions AS A
INNER JOIN jsql_test.clients AS C
  ON A.client_id = C.id
INNER JOIN jsql_test.plans AS P
  ON A.plan_id = P.id
GROUP BY C._source->>'city'
HAVING COUNT(A.id)::bigint > 1
ORDER BY C._source->>'city' ASC
LIMIT 1000;

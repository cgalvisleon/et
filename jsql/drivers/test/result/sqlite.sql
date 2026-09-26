-- TestInsertUpdate · DDL
CREATE TABLE IF NOT EXISTS transfers (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  kind TEXT DEFAULT NULL,
  code TEXT DEFAULT NULL,
  client_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS transfers_status_idx ON transfers (status);
CREATE INDEX IF NOT EXISTS transfers__idx_idx ON transfers (_idx);

-- TestInsertUpdate · INSERT
INSERT INTO transfers
  (_idx, client_id, code, id, kind, status, _source)
VALUES
  ('1790443470728', '1000000001', '00000009', '1e0fffc3-448a-47b8-92e1-97c572ba388c', 'transfers', 'en_process', '{"app_id":"Vista360","appointmentDate":"2026-07-24","billingAddress":null,"caption":"Traslate_Order 00000009","channel":"","createdAt":"2026-07-23T20:06:44.670Z","data":{"appointment_date":"2026-07-24","create_automatic_ticket":true,"extended_attribute_values":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extended_attributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"freshdeskTicketId":100001,"plans_result":null,"selected_account":"10000000001","selected_plan":null,"selected_service":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"stratum_change_attachments":[],"transfer_address":{"address":"CR 2 CL 3-04 APT1","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"department_name":"VALLE DEL CAUCA","depto_id":"76","formatted_address":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","municipality_id":"76520","municipality_name":"PALMIRA","neighborhood_id":3009,"neighborhood_name":"RECREO","population_center":"","property_type_id":1,"property_type_name":"RESIDENCIAL","stratum":"4"},"transfer_address_form":{"department":{"attribute":"department","id":"b9cc5d75-5839-402a-bb2d-dbddbccd8948","idCode":"76","idTable":"24","label":"VALLE DEL CAUCA","name":"VALLE DEL CAUCA","value":"VALLE DEL CAUCA"},"municipality":{"attribute":"municipality","id":"7d58f5a1-964e-42c1-bf70-55e20244957b","idCode":"76520","idTable":"1032","label":"PALMIRA","name":"PALMIRA","value":"PALMIRA"},"neighborhood":{"attribute":"neighborhood","id":3009,"label":"RECREO","name":"RECREO","value":"RECREO"},"numberPrimary":"2","numberSecondary":"3","observations":"","plate":"04 APT1","propertyType":{"id":1,"name":"RESIDENCIAL"},"routePrimary":{"id":"CR","label":"CARRERA","name":"CR"},"routeSecondary":{"id":"CL","label":"CALLE","name":"CL"},"stratum":{"id":"4","name":"4"}},"transfer_address_map_selection":{"accuracy":"approximate","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"formattedAddress":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","source":"address"},"transfer_address_preview":"CR 2 CL 3-04 APT1","transfer_address_validation":{"cycle":"","message":"Reserva obtenida por dirección.","reserved_port_id":"100002","selected_stratum":"4","suggested_stratum":"-","typeOfService":"SERVICIO FISICO"},"transfer_resumed_from_history":true,"type_of_service_sugerido":"SERVICIO FISICO"},"description":"Traslate_Order 00000009","endDate":"","extendedAttributeValues":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extendedAttributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"maxStepReached":3,"orderCode":"00000009","payload":null,"plansResult":null,"project_id":"-1","selectedAccount":"10000000001","selectedAvailabilityDate":"","selectedAvailabilitySlot":null,"selectedPlan":null,"selectedService":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"serviceId":"1e0fffc3-448a-47b8-92e1-97c572ba388c","step":3,"transferAddress":null}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."kind"', kind,
'$."code"', code,
'$."client_id"', client_id
) AS result;

-- TestInsertUpdate · SQL
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."kind"', A.kind,
'$."code"', A.code,
'$."client_id"', A.client_id
) AS result
FROM transfers AS A
WHERE A.id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
LIMIT 1;

-- TestInsertUpdate · SQL
SELECT
json_object(
'id', A.id,
'code', A.code,
'municipality', json(A._source -> '$."data"."transfer_address"."municipality_name"'),
'plan', json(A._source -> '$."extendedAttributeValues"."plan_comercial"'),
'step', json(A._source -> '$."step"')
) AS result
FROM transfers AS A
WHERE A.kind = 'transfers'
  AND json_extract(A._source, '$."data"."transfer_address"."stratum"') = '4'
  AND json_extract(A._source, '$."step"') = 3
LIMIT 1000;

-- TestInsertUpdate · SQL
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."kind"', A.kind,
'$."code"', A.code,
'$."client_id"', A.client_id
) AS result
FROM transfers AS A
WHERE A.id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
LIMIT 1000;

-- TestInsertUpdate · UPDATE
UPDATE transfers
SET
  client_id = '1000000001',
  code = '00000010',
  created_at = NULL,
  kind = 'transfers',
  status = 'done',
  updated_at = NULL,
  _source = json_set(COALESCE(_source, '{}'),
'$."app_id"', json('"Vista360"'),
'$."appointmentDate"', json('"2026-07-24"'),
'$."billingAddress"', json('null'),
'$."caption"', json('"Traslado \"urgente\" de O''Brien"'),
'$."channel"', json('""'),
'$."createdAt"', json('"2026-07-23T20:06:44.670Z"'),
'$."data"', json('{"appointment_date":"2026-07-24","create_automatic_ticket":true,"extended_attribute_values":{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"},"extended_attributes":{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}},"freshdeskTicketId":100001,"plans_result":null,"selected_account":"10000000001","selected_plan":null,"selected_service":{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"},"stratum_change_attachments":[],"transfer_address":{"address":"CR 2 CL 3-04 APT1","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"department_name":"VALLE DEL CAUCA","depto_id":"76","formatted_address":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","municipality_id":"76520","municipality_name":"PALMIRA","neighborhood_id":3009,"neighborhood_name":"RECREO","population_center":"","property_type_id":1,"property_type_name":"RESIDENCIAL","stratum":"4"},"transfer_address_form":{"department":{"attribute":"department","id":"b9cc5d75-5839-402a-bb2d-dbddbccd8948","idCode":"76","idTable":"24","label":"VALLE DEL CAUCA","name":"VALLE DEL CAUCA","value":"VALLE DEL CAUCA"},"municipality":{"attribute":"municipality","id":"7d58f5a1-964e-42c1-bf70-55e20244957b","idCode":"76520","idTable":"1032","label":"PALMIRA","name":"PALMIRA","value":"PALMIRA"},"neighborhood":{"attribute":"neighborhood","id":3009,"label":"RECREO","name":"RECREO","value":"RECREO"},"numberPrimary":"2","numberSecondary":"3","observations":"","plate":"04 APT1","propertyType":{"id":1,"name":"RESIDENCIAL"},"routePrimary":{"id":"CR","label":"CARRERA","name":"CR"},"routeSecondary":{"id":"CL","label":"CALLE","name":"CL"},"stratum":{"id":"4","name":"4"}},"transfer_address_map_selection":{"accuracy":"approximate","coordinates":{"lat":3.1234567,"lng":-76.12345678901234},"formattedAddress":"Cra. 2 # 3-04, Palmira, Valle del Cauca, Colombia","source":"address"},"transfer_address_preview":"CR 2 CL 3-04 APT1","transfer_address_validation":{"cycle":"","message":"Reserva obtenida por dirección.","reserved_port_id":"100002","selected_stratum":"4","suggested_stratum":"-","typeOfService":"SERVICIO FISICO"},"transfer_resumed_from_history":true,"type_of_service_sugerido":"SERVICIO FISICO"}'),
'$."data"."selected_service"."status"', json('"inactivo"'),
'$."data"."transfer_address"."notes"', json('"C:\\ruta\\nueva"'),
'$."data"."transfer_address"."stratum"', json('"5"'),
'$."description"', json('"Traslate_Order 00000009"'),
'$."endDate"', json('""'),
'$."extendedAttributeValues"', json('{"canal_venta":"VENTA EXTERNA","cliente_tiene_ont":"NO","correo_electronico":"cliente@example.com","id_cuenta":"10000000001","nombre":"Cliente De Prueba","numero_documento":"1000000001","observaciones":"NA","operador_actual":"CLARO","otro_operador":"","otro_telefono":"","plan_comercial":"PLAN 200 MEGAS 2026 F","telefono_movil":"3000000000","tipo_cliente":"PROPIETARIO","tipo_de_instalacion":"","tipo_documento":"C.C","tipo_uso_estratos":"2"}'),
'$."extendedAttributeValues"."tipo_cliente"', json('"INQUILINO"'),
'$."extendedAttributes"', json('{"canal_venta":{"ATTRID":"301","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"CONDITIONAL_LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"[{\"value\": \"VENTA EXTERNA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PUNTO DE VENTA\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"UNIDADES RESIDENCIALES\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"PROYECTOS DE CONEXIÓN\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"TELEFÓNICO\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"WHATSAPP\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}},{\"value\": \"E-COMMERCE\", \"fieldCondition\": {\"203\": \"\", \"204\": \"\"}}]","NAME":"CANAL DE VENTA","POSITION":"15","POSITIONAPP":"15","SHORTCODE":"canal_venta","VALUE":"VENTA EXTERNA"},"cliente_tiene_ont":{"ATTRID":"1664","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"Y","ISMANDATORY":"Y","ISVISIBLEAPP":"Y","LISTVALUES":"SI,NO","NAME":"CLIENTE TIENE ONT","POSITION":"20","SHORTCODE":"cliente_tiene_ont"},"correo_electronico":{"ATTRID":"15","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","DESCRIPTION":"Correo electrónico","EXPRESSION":"^[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&''*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"CORREO ELECTRÓNICO","POSITION":"9","POSITIONAPP":"9","SHORTCODE":"correo_electronico","VALUE":"cliente@example.com"},"id_cuenta":{"ATTRID":"16","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Entre 1 y 11 números","EXPRESSION":"^[0-9]{1,11}","FIELDLENGTH":"11","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"ID_CUENTA","POSITION":"2","POSITIONAPP":"2","SHORTCODE":"id_cuenta","VALUE":"10000000001"},"nombre":{"ATTRID":"11","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Solo se permite letras y números","EXPRESSION":"^[A-Za-zÁÉÍÓÚÜÑáéíóúüñ0-9 ]{1,200}$","FIELDLENGTH":"200","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NOMBRE","POSITION":"3","POSITIONAPP":"3","SHORTCODE":"nombre","VALUE":"Cliente De Prueba"},"numero_documento":{"ATTRID":"34","CATEGORYNAME":"CLIENTE","DATATYPE":"VARCHAR","DESCRIPTION":"Sólo números sin espacios","EXPRESSION":"^[0-9]*$","FIELDLENGTH":"12","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"NÚMERO DE DOCUMENTO","POSITION":"5","POSITIONAPP":"3","SHORTCODE":"numero_documento","VALUE":"1000000001"},"observaciones":{"ATTRID":"56","CATEGORYNAME":"OPORTUNIDAD","DATATYPE":"VARCHAR","DEFAULTVALUE":"NA","EXPRESSION":"","FIELDLENGTH":"2000","ISEDITABLE":"Y","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"OBSERVACIONES","POSITION":"500","POSITIONAPP":"19","SHORTCODE":"observaciones"},"operador_actual":{"ATTRID":"302","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"CLARO,MOVISTAR,TIGO,DIRECTV,LEGON,ESG,INTERMAX,CABLENET,COLOMBIATEL,CABLE CAUCA,TELECABLE,INTERCOM,REDESMAS,PLUSNET,VALLETEL,SERVINET,BITWAN,SUPER REDES,ERT,FIBERNET,OTRO,HUGHESNET,CABLEFUTURO,CLAN,MAXTV,NINGUNO","NAME":"OPERADOR ACTUAL","POSITION":"17","POSITIONAPP":"17","SHORTCODE":"operador_actual","VALUE":"CLARO"},"otro_operador":{"ATTRID":"303","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VARCHAR","EXPRESSION":"","FIELDLENGTH":"20","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"2 OTRO OPERADOR","POSITION":"18","POSITIONAPP":"18","SHORTCODE":"otro_operador"},"otro_telefono":{"ATTRID":"0","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELEFONO ALTERNATIVO","POSITION":"8","POSITIONAPP":"8","SHORTCODE":"otro_telefono"},"plan_comercial":{"ADDITIONAL_SERVICES":"","ATTRID":"117","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"VIEW","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"Y","LISTVALUES":"{\"view\": \"planModal\",\"es\":{\"stratum\": \"TIPO DE USO ESTRATO\",\"municipality\": \"CIUDAD\",\"identificationNumber\":\"NÚMERO DE DOCUMENTO\"},\"en\": {\"stratum\": \"Type of use stratum\",\"municipality\": \"CITY\",\"identificationNumber\":\"Document number\"}}","MODULE":"inventory_catalog_services","NAME":"PLAN COMERCIAL","POSITION":"351","SHORTCODE":"plan_comercial","SHOWCOLUMNNAME":"name","VALUE":"PLAN 200 MEGAS 2026 F"},"telefono_movil":{"ATTRID":"40","CATEGORYNAME":"TOMADOR SERVICIO","DATATYPE":"VARCHAR","DESCRIPTION":"Número de 10 dígitos","EXPRESSION":"^\\d{10}$","FIELDLENGTH":"10","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","NAME":"TELÉFONO MÓVIL","POSITION":"7","POSITIONAPP":"8","SHORTCODE":"telefono_movil","VALUE":"3000000000"},"tipo_cliente":{"ATTRID":"300","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"INQUILINO,PROPIETARIO","NAME":"TIPO DE CLIENTE","POSITION":"1","POSITIONAPP":"1","SHORTCODE":"tipo_cliente","VALUE":"PROPIETARIO"},"tipo_de_instalacion":{"ATTRID":"1663","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"SERVICIO FISICO,SERVICIO REMOTO","NAME":"TIPO DE INSTALACIÓN","POSITION":"100","POSITIONAPP":"100","SHORTCODE":"tipo_de_instalacion"},"tipo_documento":{"ATTRID":"33","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"C.C","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"C.C,NIT,C.E,PPT","NAME":"TIPO DE DOCUMENTO","POSITION":"4","POSITIONAPP":"2","SHORTCODE":"tipo_documento","VALUE":"C.C"},"tipo_uso_estratos":{"ATTRID":"7","CATEGORYNAME":"SIN CATEGORIZAR","DATATYPE":"LIST","DEFAULTVALUE":"N/A","EXPRESSION":"","ISEDITABLE":"N","ISMANDATORY":"N","ISVISIBLEAPP":"N","LISTVALUES":"1,2,3,4,5,6,COMERCIAL,OFICIAL,AUTOCONSUMO","NAME":"TIPO DE USO ESTRATO","POSITION":"6","POSITIONAPP":"8","SHORTCODE":"tipo_uso_estratos","VALUE":"2"}}'),
'$."maxStepReached"', json('3'),
'$."orderCode"', json('"00000009"'),
'$."payload"', json('null'),
'$."plansResult"', json('null'),
'$."project_id"', json('"-1"'),
'$."selectedAccount"', json('"10000000001"'),
'$."selectedAvailabilityDate"', json('""'),
'$."selectedAvailabilitySlot"', json('null'),
'$."selectedPlan"', json('null'),
'$."selectedService"', json('{"account":"10000000001","address":{"address":"CL 1 CR 1-01","city":"PALMIRA","department":"VALLE DEL CAUCA","neighborhood":"ROZO"},"address_invoice":{"address":"","city":"","department":"","neighborhood":""},"id":"10000000001","payments":[],"plan":"PLAN 200 MEGAS 2026 F","speed":"","status":"activo","termination_date":"0001-01-01T00:00:00Z","vinculation_date":"0001-01-01T00:00:00Z"}'),
'$."serviceId"', json('"1e0fffc3-448a-47b8-92e1-97c572ba388c"'),
'$."step"', json('4'),
'$."transferAddress"', json('null')
)
WHERE id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."kind"', kind,
'$."code"', code,
'$."client_id"', client_id
) AS result;

-- TestInsertUpdate · SQL
SELECT
json_set(json_remove(COALESCE(A._source, '{}'), '$."_idx"'),
'$."created_at"', A.created_at,
'$."updated_at"', A.updated_at,
'$."status"', A.status,
'$."id"', A.id,
'$."kind"', A.kind,
'$."code"', A.code,
'$."client_id"', A.client_id
) AS result
FROM transfers AS A
WHERE A.id = '1e0fffc3-448a-47b8-92e1-97c572ba388c'
LIMIT 1;

-- TestJoinGroupHaving · DDL
CREATE TABLE IF NOT EXISTS clients (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS clients_status_idx ON clients (status);
CREATE INDEX IF NOT EXISTS clients__idx_idx ON clients (_idx);

-- TestJoinGroupHaving · DDL
CREATE TABLE IF NOT EXISTS plans (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  name TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS plans_status_idx ON plans (status);
CREATE INDEX IF NOT EXISTS plans__idx_idx ON plans (_idx);

-- TestJoinGroupHaving · DDL
CREATE TABLE IF NOT EXISTS subscriptions (
  created_at TEXT DEFAULT NULL,
  updated_at TEXT DEFAULT NULL,
  status TEXT DEFAULT 'active',
  id TEXT DEFAULT NULL,
  _source TEXT DEFAULT '{}',
  client_id TEXT DEFAULT NULL,
  plan_id TEXT DEFAULT NULL,
  _idx TEXT DEFAULT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS subscriptions_status_idx ON subscriptions (status);
CREATE INDEX IF NOT EXISTS subscriptions__idx_idx ON subscriptions (_idx);

-- TestJoinGroupHaving · INSERT
INSERT INTO clients
  (_idx, id, name, _source)
VALUES
  ('1790443471404', 'c1', 'Ana', '{"city":"PALMIRA"}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO clients
  (_idx, id, name, _source)
VALUES
  ('1790443471404', 'c2', 'Luis', '{"city":"CALI"}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO clients
  (_idx, id, name, _source)
VALUES
  ('1790443471404', 'c3', 'Marta O''Neil', '{"city":"PALMIRA"}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO plans
  (_idx, id, name, _source)
VALUES
  ('1790443471405', 'p1', 'PLAN 200', '{"price":90000}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO plans
  (_idx, id, name, _source)
VALUES
  ('1790443471405', 'p2', 'PLAN 500', '{"price":150000}')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."name"', name
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471406', 'c1', 's1', 'p1')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."client_id"', client_id,
'$."plan_id"', plan_id
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471406', 'c1', 's2', 'p2')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."client_id"', client_id,
'$."plan_id"', plan_id
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471406', 'c2', 's3', 'p1')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."client_id"', client_id,
'$."plan_id"', plan_id
) AS result;

-- TestJoinGroupHaving · INSERT
INSERT INTO subscriptions
  (_idx, client_id, id, plan_id)
VALUES
  ('1790443471407', 'c3', 's4', 'p2')
RETURNING json_set(json_remove(COALESCE(_source, '{}'), '$."_idx"'),
'$."created_at"', created_at,
'$."updated_at"', updated_at,
'$."status"', status,
'$."id"', id,
'$."client_id"', client_id,
'$."plan_id"', plan_id
) AS result;

-- TestJoinGroupHaving · SQL
SELECT
json_object(
'id', A.id,
'client', C.name,
'city', json(C._source -> '$."city"'),
'plan', P.name,
'price', json(P._source -> '$."price"')
) AS result
FROM subscriptions AS A
INNER JOIN clients AS C
  ON A.client_id = C.id
INNER JOIN plans AS P
  ON A.plan_id = P.id
ORDER BY A.id ASC
LIMIT 1000;

-- TestJoinGroupHaving · SQL
SELECT
json_object(
'city', json(C._source -> '$."city"'),
'total', COUNT(A.id),
'amount', COALESCE(SUM(json_extract(P._source, '$."price"')), 0)
) AS result
FROM subscriptions AS A
INNER JOIN clients AS C
  ON A.client_id = C.id
INNER JOIN plans AS P
  ON A.plan_id = P.id
GROUP BY json_extract(C._source, '$."city"')
ORDER BY json_extract(C._source, '$."city"') ASC
LIMIT 1000;

-- TestJoinGroupHaving · SQL
SELECT
json_object(
'city', json(C._source -> '$."city"'),
'total', COUNT(A.id),
'amount', COALESCE(SUM(json_extract(P._source, '$."price"')), 0)
) AS result
FROM subscriptions AS A
INNER JOIN clients AS C
  ON A.client_id = C.id
INNER JOIN plans AS P
  ON A.plan_id = P.id
GROUP BY json_extract(C._source, '$."city"')
HAVING COUNT(A.id) > 1
ORDER BY json_extract(C._source, '$."city"') ASC
LIMIT 1000;

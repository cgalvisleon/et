## WebSocket

Crea un servidor de WebSocket usando la implementación de la librería Gorilla. Debe permitir conectarse, suscribirse, suscribirse de tipo queue, desuscribirse y publicar mensajes en un canal, tanto desde el servidor como desde un cliente. En una suscripción normal (topic) el cliente recibe todos los mensajes del canal; en una de tipo queue los clientes se agrupan y cada mensaje lo recibe un solo miembro del grupo, por turnos.

Debe tener los eventos OnConnect, OnDisconnect, OnSubscribe y OnUnsubscribe para ampliar sus capacidades. OnConnect y OnSubscribe pueden rechazar la conexión o la suscripción, por ejemplo para validar un token o permisos; OnDisconnect y OnUnsubscribe avisan cuando un cliente se va o deja un canal, también cuando se pierde la conexión.

El servidor se puede montar en cualquier router o escuchar por su cuenta, y un cliente lento no debe frenar a los demás.

Crea también un cliente para el servidor, con su NewClient y su Connect, que devuelva un Client. El cliente se suscribe, se suscribe de tipo queue, se desuscribe y publica esperando la confirmación del servidor, recibe los mensajes en orden con OnMessage y OnDirect, y tiene OnConnect y OnDisconnect para poder reconectarse. Si el servidor rechaza la conexión, Connect devuelve el motivo. Del lado del servidor, cada cliente conectado se representa con Conn.

Usa el modelo catalog para agrupar en catalog.go los wrappers de los métodos públicos.

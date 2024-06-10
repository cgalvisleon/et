package ws

const (
	MSG_CLIENT_CONNECT          = "Client connected %s to Hub:%s"
	MSG_CLIENT_DISCONNECT       = "Client disconnected %s to Hub:%s"
	ERR_CLIENT_NOT_FOUND        = "Client not found"
	ERR_CLIENT_IS_CLOSED        = "Client is closed"
	ERR_NOT_WS_SERVICE          = "Not websocket service"
	ERR_NOT_DEFINE_CLIENTID     = "Not define clientId"
	ERR_NOT_DEFINE_USERNAME     = "Not define username"
	ERR_NOT_CONNECT_WS          = "Not connect socket"
	ERR_CHANNEL_NOT_FOUND       = "Channel not found"
	ERR_CHANNEL_NOT_SUBSCRIBERS = "Channel not subscribers"
	ERR_CHANNEL_EMPTY           = "Channel is empty"
	ERR_PARAM_NOT_FOUND         = "Param not found"
	ERR_CLIENT_ID_EMPTY         = "Client id is empty"
	ERR_MESSAGE_UNFORMATTED     = "Message unformatted"
	ERR_REDISADAPTER_NOT_FOUND  = "Redis adapter not found"
	PARAMS_UPDATED              = "Params updated"
)

package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/response"
)

/**
* HttpGetPublications
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpDescribe(w http.ResponseWriter, r *http.Request) {
	result := h.Describe()

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HttpGetPublications
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpGetPublications(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	name := query.Str("name")
	queue := query.Str("queue")
	items := h.GetChannels(name, queue)

	response.ITEMS(w, r, http.StatusOK, items)
}

/**
* HttpGetSubscribers
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpGetSubscribers(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")
	items := h.GetClients(key)

	response.ITEMS(w, r, http.StatusOK, items)
}

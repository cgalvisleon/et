package event

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

const (
	Post = "POST"
)

type Router interface {
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
}

func LoadRouter(r Router) {
	r.Protect(Post, "/events/publish", HttpEventPublish)
}

/**
* HttpEventPublish
* @param w http.ResponseWriter, r *http.Request
**/
func HttpEventPublish(w http.ResponseWriter, r *http.Request) {
	body, _ := request.GetBody(r)
	channel := body.Str("channel")
	data := body.Json("data")
	err := Publish(channel, data)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, et.Item{
		Ok: err == nil,
		Result: et.Json{
			"message": "Event published",
		},
	})
}

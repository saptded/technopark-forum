package delivery

import (
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"net/http"
	"technopark-forum/models"
	"technopark-forum/usecase"
)

type Api struct {
	usecase *usecase.Service
}

func NewApi(usecase *usecase.Service) *Api {
	return &Api{usecase: usecase}
}

func (api *Api) CreateUser(ctx *fasthttp.RequestCtx) {
	var user models.User

	_ = user.UnmarshalJSON(ctx.PostBody())
	user.Nickname = ctx.UserValue("nickname").(string)

	users, err := api.usecase.CreateUser(&user)
	if err != nil {
		ctx.Error(err.Error(), http.StatusInternalServerError)
	}

	var response []byte
	if users != nil {
		ctx.SetStatusCode(http.StatusConflict)
		response, _ = easyjson.Marshal(users)
	} else {
		ctx.SetStatusCode(http.StatusCreated)
		response, _ = easyjson.Marshal(user)
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

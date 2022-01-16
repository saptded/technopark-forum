package delivery

import (
	"encoding/json"
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
	user := new(models.User)
	_ = easyjson.Unmarshal(ctx.PostBody(), user)
	user.Nickname = ctx.UserValue("nickname").(string)

	users, err := api.usecase.CreateUser(user)
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

func (api *Api) GetUserProfile(ctx *fasthttp.RequestCtx) {
	nickname := ctx.UserValue("nickname").(string)

	user, err := api.usecase.GetUserProfile(nickname)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusOK)
		response, _ = easyjson.Marshal(user)
	} else if err.Error() == models.UserNotFound(nickname).Error() {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) UpdateUserProfile(ctx *fasthttp.RequestCtx) {
	user := new(models.User)
	_ = easyjson.Unmarshal(ctx.PostBody(), user)
	user.Nickname = ctx.UserValue("nickname").(string)

	newUser, err := api.usecase.UpdateUserProfile(user)
	var response []byte

	if err == nil {
		ctx.SetStatusCode(http.StatusOK)
		response, _ = easyjson.Marshal(newUser)
	} else {
		switch {
		case models.UserNotFound(user.Nickname).Error() == err.Error():
			ctx.SetStatusCode(http.StatusNotFound)
			response, _ = json.Marshal(err)
		case models.UsersProfileConflict(user.Nickname).Error() == err.Error():
			ctx.SetStatusCode(http.StatusConflict)
			response, _ = json.Marshal(err)
		}
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

package delivery

import (
	"encoding/json"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
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

// service

func (api *Api) GetStatus(ctx *fasthttp.RequestCtx) {
	status, err := api.usecase.GetStatus()

	var response []byte
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	} else {
		ctx.SetStatusCode(http.StatusOK)
		response, _ = easyjson.Marshal(status)
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) Clear(ctx *fasthttp.RequestCtx) {
	err := api.usecase.Clear()

	var response []byte
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	} else {
		ctx.SetStatusCode(http.StatusOK)
		response, _ = easyjson.Marshal(models.ErrorMessage(errors.New("cleared")))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

// user

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

// forum

func (api *Api) CreateForum(ctx *fasthttp.RequestCtx) {
	forum := new(models.Forum)
	_ = easyjson.Unmarshal(ctx.PostBody(), forum)
	author := forum.Author

	forum, err := api.usecase.CreateForum(forum)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusCreated)
		response, _ = easyjson.Marshal(forum)
	} else if forum != nil && err != nil {
		ctx.SetStatusCode(http.StatusConflict)
		response, _ = easyjson.Marshal(forum)
	} else if err.Error() == models.UserNotFound(author).Error() {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) GetForum(ctx *fasthttp.RequestCtx) {
	slug := ctx.UserValue("slug").(string)

	forum, err := api.usecase.GetForum(slug)

	var response []byte
	if err == nil {
		response, _ = easyjson.Marshal(forum)
	} else {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) CreateThread(ctx *fasthttp.RequestCtx) {
	thread := new(models.Thread)
	_ = easyjson.Unmarshal(ctx.PostBody(), thread)

	slug := ctx.UserValue("slug").(string)
	thread.Forum = slug

	gotThread, err := api.usecase.CreateThread(slug, thread)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusCreated)
		response, _ = easyjson.Marshal(gotThread)
	} else if models.UserNotFound(thread.Author).Error() == err.Error() ||
		models.ForumNotFound(thread.Slug).Error() == err.Error() {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	} else if err == models.Conflict {
		ctx.SetStatusCode(http.StatusConflict)
		response, _ = easyjson.Marshal(gotThread)
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) GetUsers(ctx *fasthttp.RequestCtx) {

	slug := ctx.UserValue("slug").(string)
	limit := ctx.QueryArgs().Peek("limit")
	desc := ctx.QueryArgs().Peek("desc")
	since := ctx.QueryArgs().Peek("since")

	users, err := api.usecase.GetForumUsers(slug, limit, since, desc)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusOK)
		if len(*users) != 0 {
			response, _ = easyjson.Marshal(users)
		} else {
			response = []byte("[]")
		}
	} else if models.ForumNotFound(slug).Error() == err.Error() {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) GetThreads(ctx *fasthttp.RequestCtx) {

	slug := ctx.UserValue("slug").(string)
	limit := ctx.QueryArgs().Peek("limit")
	desc := ctx.QueryArgs().Peek("desc")
	since := ctx.QueryArgs().Peek("since")

	threads, err := api.usecase.GetForumThreads(slug, limit, since, desc)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusOK)
		if len(*threads) != 0 {
			response, _ = easyjson.Marshal(threads)
		} else {
			response = []byte("[]")
		}
	} else if models.ForumNotFound(slug).Error() == err.Error() {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) CreatePosts(ctx *fasthttp.RequestCtx) {
	slugOrID := ctx.UserValue("slug_or_id")

	posts := models.Posts{}
	_ = easyjson.Unmarshal(ctx.PostBody(), &posts)

	newPosts, err := api.usecase.CreatePosts(slugOrID, &posts)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusCreated)
		if newPosts != nil {
			response, _ = easyjson.Marshal(newPosts)
		} else {
			response = []byte("[]")
		}
	} else if err == models.ThreadNotFound || err == models.UserNotFoundSimple {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	} else if err == models.Conflict {
		ctx.SetStatusCode(http.StatusConflict)
		response, _ = easyjson.Marshal(models.ErrorMessage(err))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) GetThread(ctx *fasthttp.RequestCtx) {
	slugOrID := ctx.UserValue("slug_or_id")

	thread, err := api.usecase.GetThread(slugOrID)

	var response []byte
	if err != nil {
		if thread == nil {
			response = []byte("[]")
		} else {
			ctx.SetStatusCode(http.StatusNotFound)
			response, _ = easyjson.Marshal(models.ErrorMessage(err))
		}
	} else {
		ctx.SetStatusCode(http.StatusOK)
		response, _ = easyjson.Marshal(thread)

	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) UpdateThread(ctx *fasthttp.RequestCtx) {
	slugOrID := ctx.UserValue("slug_or_id").(string)

	threadUpd := new(models.ThreadUpdate)
	_ = easyjson.Unmarshal(ctx.PostBody(), threadUpd)

	thread, err := api.usecase.UpdateThread(slugOrID, threadUpd)

	var response []byte
	if err == nil {
		ctx.SetStatusCode(http.StatusOK)
		response, _ = easyjson.Marshal(thread)
	} else {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ = easyjson.Marshal(models.ErrorMessage(models.ThreadNotFound))
	}

	ctx.SetContentType("application/json")
	_, _ = ctx.Write(response)
}

func (api *Api) GetPosts(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")

	slugOrID := ctx.UserValue("slug_or_id").(string)
	limit := ctx.QueryArgs().Peek("limit")
	since := ctx.QueryArgs().Peek("since")
	sort := ctx.QueryArgs().Peek("sort")
	desc := ctx.QueryArgs().Peek("desc")

	posts, statusCode := api.usecase.GetThreadPosts(&slugOrID, limit, since, sort, desc)

	ctx.SetStatusCode(statusCode)

	switch statusCode {
	case http.StatusOK:
		if len(*posts) != 0 {
			response, _ := easyjson.Marshal(posts)
			_, _ = ctx.Write(response)
		} else {
			_, _ = ctx.Write([]byte("[]"))
		}
	case http.StatusNotFound:
		response, _ := easyjson.Marshal(models.ErrorMessage(models.ThreadNotFound))
		_, _ = ctx.Write(response)
	case http.StatusInternalServerError:
		response, _ := easyjson.Marshal(models.ErrorMessage(models.Conflict))
		_, _ = ctx.Write(response)
	}
}

func (api *Api) Vote(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	vote := new(models.Vote)
	_ = easyjson.Unmarshal(ctx.PostBody(), vote)

	slugOrID := ctx.UserValue("slug_or_id")

	thread, err := api.usecase.PutVote(slugOrID, vote)
	if err != nil {
		ctx.SetStatusCode(http.StatusNotFound)
		response, _ := json.Marshal(err)
		ctx.SetContentType("application/json")
		_, _ = ctx.Write(response)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	response, _ := thread.MarshalJSON()
	_, _ = ctx.Write(response)
}

func (api *Api) GetPostDetails(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	id := ctx.UserValue("id").(string)
	related := ctx.QueryArgs().Peek("related")

	postDetails, statusCode := api.usecase.GetPostDetails(&id, related)
	ctx.SetStatusCode(statusCode)

	switch statusCode {
	case http.StatusOK:
		response, _ := easyjson.Marshal(postDetails)
		_, _ = ctx.Write(response)
	case http.StatusNotFound:
		response, _ := easyjson.Marshal(models.ErrorMessage(models.PostNotFound))
		ctx.SetContentType("application/json")
		_, _ = ctx.Write(response)
	}
}

func (api *Api) UpdatePost(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	id := ctx.UserValue("id").(string)
	postUpd := new(models.PostUpdate)

	_ = easyjson.Unmarshal(ctx.PostBody(), postUpd)
	post, statusCode := api.usecase.UpdatePostDetails(&id, postUpd)
	ctx.SetStatusCode(statusCode)

	switch statusCode {
	case http.StatusOK:
		response, _ := easyjson.Marshal(post)
		_, _ = ctx.Write(response)
	case http.StatusNotFound:
		response, _ := easyjson.Marshal(models.ErrorMessage(models.PostNotFound))
		ctx.SetContentType("application/json")
		_, _ = ctx.Write(response)
	}
}

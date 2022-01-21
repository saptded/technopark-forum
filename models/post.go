package models

import "time"

//easyjson:json
type Post struct {
	ID       int       `json:"id"`
	Author   string    `json:"author"`
	Message  string    `json:"message"`
	IsEdited bool      `json:"isEdited"`
	Forum    string    `json:"forum"`
	Thread   int       `json:"thread"`
	Created  time.Time `json:"created,omitempty"`
	Parent   int32     `json:"parent,omitempty"`
	Parents  []int32   `json:"parents"`
}

//easyjson:json
type Posts []Post

//easyjson:json
type PostDetails struct {
	AuthorDetails *User   `json:"author,omitempty"`
	ForumDetails  *Forum  `json:"forum,omitempty"`
	PostDetails   *Post   `json:"post,omitempty"`
	ThreadDetails *Thread `json:"thread,omitempty"`
}

//easyjson:json
type PostUpdate struct {
	Message *string `json:"message"`
}

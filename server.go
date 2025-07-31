package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/flosch/pongo2"
	"github.com/labstack/echo/v4"
	"yana.go/yana"
)

const IS_DEBUG = false

const USER_ID_COOKIE_NAME = "user"

type Renderer struct {
	Debug bool
}

// ------------ MISC. ------------

func (renderer Renderer) Render(writer io.Writer, site string, data interface{}, c echo.Context) error {
	var context pongo2.Context
	if data != nil {
		var ok bool
		context, ok = data.(pongo2.Context)
		if !ok {
			fmt.Println("not ok renderer (argh..): ", data)
			return errors.New("Pongo2.Context is empty")
		}
	}
	context["version"] = "V0.0.1"

	var template *pongo2.Template
	var err error
	template, err = pongo2.FromFile(site)
	if err != nil {
		fmt.Printf("err != nil in renderer (argh..): '%w'\n", err)
		fmt.Println(err)
		return err
	}
	fmt.Println("NEW REQUEST")

	return template.ExecuteWriter(context, writer)
}

func isLoggedIn(context echo.Context) bool {
	cookie, err := context.Cookie(USER_ID_COOKIE_NAME)
	return err == nil && cookie.Value != ""
}

// ------------ GET ------------

func getIndex(context echo.Context) error {
	if !isLoggedIn(context) {
		return context.Redirect(http.StatusMovedPermanently, "/welcome")
	}
	// Not handling err because:
	// 1. The cookie has already been checked in isLoggedIn()
	// 2. index.html can deal with notes being empty
	cookie, err := context.Cookie(USER_ID_COOKIE_NAME)
	if err != nil {
		fmt.Println("Error in /index:", err)
	}
	notes, err := yana.GetAllNotesOfUser(cookie.Value)
	if err != nil {
		fmt.Println("Error in /index:", err)
	}
	return context.Render(200, "static/index.html", pongo2.Context{"notes": notes, "noNotes": len(notes) == 0})
}

func getRoot(context echo.Context) error {
	if !isLoggedIn(context) {
		return context.Redirect(http.StatusMovedPermanently, "/welcome")
	}
	return context.Redirect(http.StatusMovedPermanently, "/index")
}

func getCreateNote(context echo.Context) error {
	if !isLoggedIn(context) {
		return context.Redirect(http.StatusMovedPermanently, "/welcome")
	}
	// noteTitle and noteContent are left empty
	return context.Render(200, "static/note.html", pongo2.Context{"isNewNote": true, "formLink": "/create-note"})
}

func addCookieToContext(context *echo.Context, name string, value string) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = value
	(*context).SetCookie(cookie)
}

func getLogin(context echo.Context) error {
	return context.Render(200, "static/login.html", pongo2.Context{})
}

func getLogout(context echo.Context) error {
	// if the user wasn't logged in before...
	if !isLoggedIn(context) {
		return context.Redirect(http.StatusMovedPermanently, "/welcome")
	}
	addCookieToContext(&context, USER_ID_COOKIE_NAME, "")
	return context.Render(200, "static/logout.html", pongo2.Context{})
}

func getRegister(context echo.Context) error {
	pongoContext := pongo2.Context{}
	if context.Request().Header.Get("error") == "DBConnectionFailure" {
		pongoContext = pongo2.Context{"error": "DBConnectionFailure"}
	} // else { ... TODO
	return context.Render(200, "static/register.html", pongoContext)
}

func getWelcome(context echo.Context) error {
	return context.Render(200, "static/welcome.html", pongo2.Context{"isLoggedIn": isLoggedIn(context)})
}

func getEditNote(context echo.Context) error {
	if !isLoggedIn(context) {
		return context.Redirect(http.StatusMovedPermanently, "/welcome")
	}
	postgresNoteId := context.QueryParam("noteId")
	if postgresNoteId == "" {
		return context.Redirect(http.StatusMovedPermanently, "/index")
	}
	note, err := yana.GetNoteFromNoteId(postgresNoteId)
	if err != nil {
		//context.Response().Header().Set("Error", "CouldNotFindNoteFromId")
		//return context.Redirect(http.StatusMovedPermanently, "/index")
	}
	// if this is not converted to a string, this creates a runtime error
	// due to invalid memory address or nil pointer dereference if there
	// is no isSuccesful parameter

	isSuccesful := context.QueryParam("isSuccesful")
	pongoContext := pongo2.Context{
		"isNewNote":   false,
		"formLink":    "/edit-note",
		"noteTitle":   note.Name,
		"noteContent": note.Content,
		"noteId":      note.PostgresId,
	}
	if err != nil {
		pongoContext["errorMessage"] = err.Error()
	}
	if isSuccesful == "true" || isSuccesful == "false" {
		pongoContext["isSuccesful"] = isSuccesful
	}
	return context.Render(200, "static/note.html", pongoContext)
}

// ------------ POST ------------

func postRegister(context echo.Context) error {
	userId, err := yana.InsertNewUserInPostgres(context.FormValue("email"), context.FormValue("name"), context.FormValue("password"))
	if err != nil {
		// TODO: Maybe implement custom errors to return here to string to tell the user what the problem was?
		context.Response().Header().Set("error", "DBConnectionFailure")
		// Return to register but with error
		return context.Redirect(http.StatusMovedPermanently, "/register")
	}
	if userId == "" {
		context.Response().Header().Set("error", "UserAlreadyExists")
		// Return to register but say that user with email already exists
		return context.Redirect(http.StatusMovedPermanently, "/register")
	}
	err = yana.NewBucket(userId)
	if err != nil {
		context.Response().Header().Set("error", "CouldNotCreateBucket")
		// Return to register but say that bucket couldn't be created
		return context.Redirect(http.StatusMovedPermanently, "/register")
	}
	addCookieToContext(&context, USER_ID_COOKIE_NAME, userId)
	return context.Redirect(http.StatusMovedPermanently, "/")
}

func postCreateNote(context echo.Context) error {
	// The user should absolutely be logged in if POST /create-note is called
	cookie, _ := context.Cookie(USER_ID_COOKIE_NAME)
	_, err := yana.NewNote(cookie.Value, context.FormValue("title"), context.FormValue("content"))
	if err != nil {
		pongoContext := pongo2.Context{
			"isNewNote":    true,
			"formLink":     "/create-note",
			"noteTitle":    context.FormValue("title"),
			"noteContent":  context.FormValue("content"),
			"isSuccesful":  "false",
			"errorMessage": err.Error(),
		}
		return context.Render(200, "static/note.html", pongoContext)
	}
	fmt.Println("err is nil")
	return context.Redirect(http.StatusMovedPermanently, "/")
}

func postLogin(context echo.Context) error {
	isOk, yanaErr := yana.IsLoginOk(context.FormValue("email"), context.FormValue("password"))
	errCodeName := "errorCodeNamePlaceholder" // TODO
	if yanaErr.Err != nil {
		switch yanaErr.Code {
		default:
			// TODO
		}
		context.Response().Header().Set("error", errCodeName)
		return context.Redirect(http.StatusMovedPermanently, "/login")
	}
	if !isOk {
		context.Response().Header().Set("error", "userDoesNotExist")
		return context.Redirect(http.StatusMovedPermanently, "/login")
	}
	userid, err := yana.GetUserIDFromEmail(context.FormValue("email"))
	if err != nil {
		context.Response().Header().Set("error", errCodeName)
		return context.Redirect(http.StatusMovedPermanently, "/login")
	}
	addCookieToContext(&context, USER_ID_COOKIE_NAME, userid)
	return context.Redirect(http.StatusMovedPermanently, "/")
}

func postEditNote(context echo.Context) error {
	/*
		I originally planed to save the original title as a hidden input in note.html because
		it is not possible to change the filename in MinIO.
		So I wanted yana.UpdateNote() to just overwrite the content if the title didn't changed.
		Problem: I realised that it isn't possible to modify the content of an existing file too
		(except for appending something to the end of a file). So yana.UpdateNote() is forced to
		create a completely new file either way.
	*/
	// The user should absolutely be logged in if POST /edit-note is called
	userId, _ := context.Cookie(USER_ID_COOKIE_NAME)
	noteId := context.FormValue("noteId")
	newTitle := context.FormValue("title")
	newContent := context.FormValue("content")
	_, err := yana.UpdateNote(userId.Value, noteId, newTitle, newContent)
	if err != nil {
		pongoContext := pongo2.Context{
			"isNewNote":    false,
			"formLink":     "/edit-note",
			"noteTitle":    newTitle,
			"noteContent":  newContent,
			"noteId":       noteId,
			"isSuccesful":  "false",
			"errorMessage": err.Error(),
		}
		return context.Render(200, "static/note.html", pongoContext)
	}
	return context.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/edit-note?noteId=%s&isSuccesful=%s", noteId, "true"))
}

// ------------ DELETE ------------

// called from index.html
func deleteDeleteNote(context echo.Context) error {
	jsonMap := make(map[string]interface{})
	err := json.NewDecoder(context.Request().Body).Decode(&jsonMap)
	if err != nil {
		fmt.Printf("Error in deleteDeleteNote: %w", err)
		fmt.Println("Should load back to root")
		return context.Redirect(http.StatusMovedPermanently, "/")
	}
	var noteId string = jsonMap["noteId"].(string)
	err = yana.DeleteNoteFromNoteId(noteId)
	if err != nil {
		fmt.Printf("Could get noteId but failed deleting note: %w", err)
	}
	fmt.Println("Should load back to index")
	return context.Redirect(http.StatusMovedPermanently, "/index")
}

func initRoutes(e *echo.Echo) {
	e.Static("/", "static")

	e.GET("/", getRoot)
	e.GET("/index", getIndex)
	e.GET("/create-note", getCreateNote)
	e.GET("/login", getLogin)
	e.GET("/register", getRegister)
	e.GET("/welcome", getWelcome)
	e.GET("/logout", getLogout)
	e.GET("/edit-note", getEditNote)

	e.POST("/login", postLogin)
	e.POST("/create-note", postCreateNote)
	e.POST("/register", postRegister)
	e.POST("/edit-note", postEditNote)

	// edit-note and delete-note are called from javascript in index.html
	// because that unfortunately makes the most sense

	e.DELETE("/delete-note", deleteDeleteNote)
}

func main() {
	renderer := Renderer{
		Debug: false,
	}

	echoServer := echo.New()
	echoServer.Renderer = renderer

	// ChatGPT generated with a few edits by me
	echoServer.HTTPErrorHandler = func(err error, context echo.Context) {
		httpError, isOk := err.(*echo.HTTPError)
		if isOk && httpError.Code == http.StatusNotFound {
			context.Redirect(http.StatusFound, "/")
			return
		}
		// Fallback to default handler for other errors
		echoServer.DefaultHTTPErrorHandler(err, context)
	}

	initRoutes(echoServer)
	echoServer.Logger.Fatal(echoServer.Start(":1323"))
}

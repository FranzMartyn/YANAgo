package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flosch/pongo2"
	"github.com/labstack/echo/v4"
	"yana.go/yana"
)

const IS_DEBUG = false

const USER_ID_COOKIE_NAME = "user"

type Renderer struct {
	Debug bool
}

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

// ------------ GET ------------

func getIndex(context echo.Context) error {
	cookie, err := context.Cookie(USER_ID_COOKIE_NAME)
	if err != nil || cookie.Value == "" {
		return context.Redirect(http.StatusMovedPermanently, "/welcome")
	}
	notes, err := yana.GetAllNotesOfUser(cookie.Value)
	if err != nil {
	}
	noNotes := false
	if len(notes) == 0 {
		noNotes = true
	}
	return context.Render(200, "static/index.html", pongo2.Context{"notes": notes, "noNotes": noNotes})
}

func getRoot(context echo.Context) error {
	return context.Redirect(http.StatusMovedPermanently, "/index")
}

func getCreateNote(context echo.Context) error {
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
	var loggedIn bool
	cookie, err := context.Cookie(USER_ID_COOKIE_NAME)
	if err != nil {
		loggedIn = false
	} else if cookie.Value == "" {
		loggedIn = false
	} else {
		loggedIn = true
	}
	return context.Render(200, "static/welcome.html", pongo2.Context{"isLoggedIn": loggedIn})
}

func getEditNote(context echo.Context) error {
	postgresNoteId := context.QueryParam("noteId")
	if postgresNoteId == "" {
		return context.Redirect(http.StatusMovedPermanently, "/index")
	}
	note, err := yana.GetNoteFromNoteId(postgresNoteId)
	if err != nil {
		context.Response().Header().Set("Error", "CouldNotFindNoteFromId")
		return context.Redirect(http.StatusMovedPermanently, "/index")
	}
	return context.Render(200, "static\\note.html", pongo2.Context{"isNewNote": false, "formLink": "/edit-note", "noteTitle": note.Name, "noteContent": note.Content, "noteId": note.PostgresId})
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
	userId, err := context.Cookie(USER_ID_COOKIE_NAME)
	if err != nil || userId.Value == "" {
		context.Response().Header().Set("error", "CouldNotCreateNote")
		return context.Redirect(http.StatusMovedPermanently, "/")
	}
	_, err = yana.NewNote(userId.Value, context.FormValue("title")+".txt", context.FormValue("content"))
	if err != nil {
		context.Redirect(http.StatusMovedPermanently, "/create-note")
	}
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
		I originally planed to save the original title as a hidden input in note.html
		because it is not possible to change the filename in MinIO so yana.UpdateNote
		just creates a new note if the title changed. Problem: I realised that it
		isn't possible to modify the content (except for appending something to
		the end of a file). So
	*/
	noteId := context.FormValue("noteId")
	noteIdAlt := context.QueryParam("noteId")
	title := context.FormValue("title")
	content := context.FormValue("content")
	userId, err := context.Cookie(USER_ID_COOKIE_NAME)
	if err != nil {
		return context.Redirect(http.StatusMovedPermanently, "/edit-note?noteId="+noteId)
	}
	updateNoteState, err := yana.UpdateNote(userId.Value, noteId, title, content)
	if err != nil {
	}
	return context.Redirect(http.StatusMovedPermanently, "/edit-note?noteId="+noteId)
}

// ------------ DELETE ------------

// called from index.html
func deleteDeleteNote(context echo.Context) error {
	fmt.Println("deleteDeleteNote called")
	return context.Redirect(http.StatusMovedPermanently, "/")
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
	initRoutes(echoServer)
	echoServer.Logger.Fatal(echoServer.Start(":1323"))
	echoServer.Logger.Info("Started at: %s\n", time.Now())
}

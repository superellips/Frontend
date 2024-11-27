package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Page struct {
	Title string
	Body  []byte
}

func loadPage(title string) (*Page, error) {
	filename := "./pages/" + title + ".html"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func getIndex(c *gin.Context) {
	p, err := loadPage("index")
	if err != nil {
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func getRegister(c *gin.Context) {
	p, err := loadPage("register")
	if err != nil {
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func postRegister(c *gin.Context) {
	data := map[string]string{"name": c.Request.FormValue("name")}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	url := "http://apigateway:8080/user/register"
	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	var returnData map[string]string
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		return
	}
	fmt.Println(returnData["id"])
	c.Redirect(http.StatusFound, "/user/"+returnData["id"])
}

func getLogin(c *gin.Context) {
	p, err := loadPage("login")
	if err != nil {
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func getUserPage(c *gin.Context) {
	id := c.Param("id")
	url := "http://apigateway:8080/user/" + id
	response, err := http.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	var returnData map[string]string
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		return
	}
	p := Page{Title: returnData["name"], Body: []byte("<a href='/guestbook/create'>Create guestbook.</a>")}
	renderTemplate(c.Writer, "default", &p)
}

func getCreateGuestbook(c *gin.Context) {
	p, err := loadPage("create_guestbook")
	if err != nil {
		return
	}
	p.Title = "Create New Guestbook"
	renderTemplate(c.Writer, "default", p)
}

func postCreateGuestbook(c *gin.Context) {

}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	t, _ := template.ParseFiles("./templates/" + tmpl + ".html")
	t.Execute(w, p)
}

func main() {
	hostname := "http://" + os.Getenv("GUESTBOOK_ROOT_DOMAIN")

	router := gin.Default()

	strictCors := cors.New(cors.Config{
		AllowOrigins:     []string{hostname},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type", "Origin"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})

	rootGroup := router.Group("/")
	rootGroup.Use(strictCors)
	{
		rootGroup.GET("/", getIndex)
		rootGroup.GET("/register", getRegister)
		rootGroup.POST("/register", postRegister)
		rootGroup.GET("/login", getLogin)
		rootGroup.GET("/user/:id", getUserPage)
		rootGroup.GET("/guestbook/create", getCreateGuestbook)
		rootGroup.POST("/guestbook/create", postCreateGuestbook)
	}

	router.Run()
}

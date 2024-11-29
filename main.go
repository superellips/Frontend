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
	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("my_secret_key")

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

func GenerateToken(username string, id string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name":   username,
		"userId": id,
		"exp":    time.Now().Add(1 * time.Hour).Unix(),
	})
	return token.SignedString(jwtKey)
}

func ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
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
	c.Redirect(http.StatusFound, "/login")
}

func getLogin(c *gin.Context) {
	p, err := loadPage("login")
	if err != nil {
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func postLogin(c *gin.Context) {
	name := c.Request.FormValue("name")
	// TODO: check a to be implemented password here
	url := "http://apigateway:8080/user/name/" + name
	response, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user data"})
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}
	var returnData map[string]string
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response data"})
		return
	}
	token, err := GenerateToken(name, returnData["id"])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}
	c.SetCookie("auth", token, 3600, "/", "http://localhost:8888", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

func getUserPage(c *gin.Context) {
	id := c.Param("id")
	url := "http://apigateway:8080/user/id/" + id
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
	tokenString, err := c.Cookie("auth")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := ValidateToken(tokenString)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	claims, _ := token.Claims.(jwt.MapClaims)
	if claims["name"] != returnData["name"] {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized."})
		return
	}
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
	tokenString, err := c.Cookie("auth")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := ValidateToken(tokenString)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	fmt.Println(claims["userId"])
	fmt.Println(c.Request.FormValue("domain"))
	fmt.Println((c.Request.FormValue("approval") == "on"))
	data := map[string]interface{}{
		"ownerId":         claims["userId"],
		"domain":          c.Request.FormValue("domain"),
		"requireApproval": (c.Request.FormValue("approval") == "on"),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	url := "http://apigateway:8080/guestbook/new"
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
	c.Redirect(http.StatusFound, "/guestbook/"+returnData["id"])
}

func getGuestbook(c *gin.Context) {

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
		rootGroup.POST("/login", postLogin)
		rootGroup.GET("/user/:id", getUserPage)
		rootGroup.GET("/guestbook/create", getCreateGuestbook)
		rootGroup.POST("/guestbook/create", postCreateGuestbook)
		rootGroup.GET("/guestbook/:id", getGuestbook)
	}

	router.Run()
}

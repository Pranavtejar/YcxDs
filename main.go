//everything in main.go
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Template struct {
	tmpl *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

type Album struct {
	Images []Image `json:"images"`
}

type Image struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type SearchResponse struct {
	Albums struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	} `json:"albums"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func getSpotifyToken(clientID, clientSecret string) string {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatal(err)
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Fatal(err)
	}

	return tokenResp.AccessToken
}

func getAlbumID(albumName, token string) string {
	albumQuery := url.QueryEscape(albumName)
	searchURL := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=album&limit=1", albumQuery)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Println("Error fetching album ID:", resp.Status)
		fmt.Println(string(data))
		log.Fatal("Failed to get album ID")
	}

	var result SearchResponse
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatal(err)
	}

	if len(result.Albums.Items) == 0 {
		log.Fatal("Album not found")
	}

	return result.Albums.Items[0].ID
}
func getAlbumCover(albumID, token string) []Image {
	var album Album

	albumURL := fmt.Sprintf("https://api.spotify.com/v1/albums/%s", albumID)
	req, err := http.NewRequest("GET", albumURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Println("Error fetching album cover:", resp.Status)
		fmt.Println(string(data))
		log.Fatal("Failed to get album cover")
	}

	if err := json.Unmarshal(data, &album); err != nil {
		log.Fatal(err)
	}

	return album.Images
}

func home(c echo.Context) error {
	albName := c.FormValue("album")
	if albName == "" {
		return c.Render(http.StatusOK, "index.html", nil)
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	token := getSpotifyToken(clientID, clientSecret)
	albumID := getAlbumID(albName, token)
	images := getAlbumCover(albumID, token)

	var imageURL string
	if len(images) > 0 {
		imageURL = images[0].URL
	}

	data := struct {
		URL string
	}{
		URL: imageURL,
	}

	return c.Render(http.StatusOK, "index.html", data)
}


func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Renderer = &Template{
		tmpl: template.Must(template.ParseGlob("templates/*.html")),
	}

	e.Any("/", home)

	e.Logger.Fatal(e.Start(":8080"))
}


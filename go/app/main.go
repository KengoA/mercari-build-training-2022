package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"image/jpeg"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "images"
	DBPath = "./db/mercari-build-training.db"
)

type Item struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
	Image    string `db:"image" json:"image"`
}

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	db, err := sql.Open("sqlite3", DBPath)
	defer db.Close()
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	items := make([]*Item, 0)
	cmd := "SELECT * FROM items;"
	rows, err := db.Query(cmd)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			c.Logger().Error(err)
			return err
		}
		items = append(items, &item)
	}

	return c.JSON(http.StatusOK, items)
}

func getItemByID(c echo.Context) error {
	ID := c.Param("id")
	db, err := sql.Open("sqlite3", DBPath)
	defer db.Close()
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	cmd := "SELECT * FROM items WHERE id = ?;"
	dbRes, err := db.Query(cmd, ID)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	var item Item
	if err := dbRes.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
		c.Logger().Error(err)
		return err
	}

	return c.JSON(http.StatusOK, item)
}

func searchItems(c echo.Context) error {
	keyword := c.QueryParam("keyword")

	if keyword == "" {
		err := fmt.Errorf("You must specify a keyword")
		c.Logger().Error(err)
		return err
	}

	c.Logger().Info(fmt.Sprintf("Searching for items including keyword: %s", keyword))

	db, err := sql.Open("sqlite3", DBPath)
	defer db.Close()
	if err != nil {
		c.Logger().Error(err)
	}

	items := make([]*Item, 0)
	cmd := "SELECT * FROM items WHERE name LIKE '%" + keyword + "%'"
	rows, err := db.Query(cmd)
	if err != nil {
		c.Logger().Error(err)
		return err
	}

	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			c.Logger().Error(err)
		}
		items = append(items, &item)
	}

	return c.JSON(http.StatusOK, items)
}

func sha256SumFromFilePath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", hash.Sum(nil)), nil
}

func imageHandler(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image, err
}

func addItem(c echo.Context) error {
	name := c.FormValue("name")
	category := c.FormValue("category")
	imagePath := c.FormValue("image")

	c.Logger().Info(fmt.Sprintf("item received, name: %s, category: %s, image path: %s", name, category, imagePath))

	image, err := imageHandler(imagePath)
	if err != nil {
		res := Response{Message: "Could not open image"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, res)
	}

	hash, err := sha256SumFromFilePath(imagePath)
	hashedFileName := hash + ".jpg"
	if err != nil {
		return err
	}

	newFile, err := os.Create(path.Join(ImgDir, hashedFileName))
	if err != nil {
		res := Response{Message: "Could not create image"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, res)
	}
	defer newFile.Close()

	options := jpeg.Options{
		Quality: 90,
	}
	err = jpeg.Encode(newFile, image, &options)
	if err != nil {
		res := Response{Message: "Could not encode new image"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, res)
	}

	c.Logger().Info(hashedFileName)

	db, err := sql.Open("sqlite3", DBPath)
	defer db.Close()

	if err != nil {
		res := Response{Message: "Could not connect to database"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadGateway, res)
	}

	cmd := "INSERT INTO items (name, category, image) VALUES (?, ?, ?)"
	_, err = db.Exec(cmd, name, category, hashedFileName)

	if err != nil {
		c.Logger().Error(err)
		return err
	}
	return nil
}

func deleteItemByID(c echo.Context) error {
	ID := c.Param("id")

	c.Logger().Info(fmt.Sprintf("deleting item with id: %s", ID))

	db, err := sql.Open("sqlite3", DBPath)
	defer db.Close()
	if err != nil {
		c.Logger().Error(err)
	}

	cmd := "DELETE FROM items WHERE id = ?"
	_, err = db.Exec(cmd, ID)

	if err != nil {
		c.Logger().Error(err)
	}
	return nil
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("itemImg"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.GET("/items", getItems)
	e.GET("/items/:id", getItemByID)
	e.GET("/search", searchItems)
	e.GET("/image/:itemImg", getImg)
	e.POST("/items", addItem)
	e.DELETE("/items/:id", deleteItemByID)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}

package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "images"
	DBPath = "../db/mercari-build-training.db"
)

type Item struct {
	ID            int    `db:"id" json:"id"`
	Name          string `db:"name" json:"name"`
	Category      string `db:"category" json:"category"`
	ImageFilename string `db:"image_filename" json:"image_filename"`
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFilename); err != nil {
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
	if err := dbRes.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFilename); err != nil {
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFilename); err != nil {
			c.Logger().Error(err)
		}
		items = append(items, &item)
	}

	return c.JSON(http.StatusOK, items)
}

func sha256SumFromString(s string) string {
	binary := sha256.Sum256([]byte(s))
	hash := hex.EncodeToString(binary[:])
	return hash
}

func addItem(c echo.Context) error {
	name := c.FormValue("name")
	category := c.FormValue("category")
	file, err := c.FormFile("image")

	// Open the file and image
	if err != nil {
		res := Response{Message: "Could not open file"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, res)
	}

	image, err := file.Open()
	if err != nil {
		res := Response{Message: "Could not read image from file"}
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, res)
	}
	defer image.Close()

	// Generate a new filename to save with a hash function
	hashedFileName := sha256SumFromString(file.Filename) + ".jpg"

	// Save the file under the images directory
	newFile, err := os.Create(path.Join(ImgDir, hashedFileName))
	if err != nil {
		res := Response{Message: "Could not create image"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, res)
	}
	defer newFile.Close()

	if _, err = io.Copy(newFile, image); err != nil {
		res := Response{Message: "Could not save new image"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, res)
	}

	// Write into the database
	db, err := sql.Open("sqlite3", DBPath)
	defer db.Close()

	if err != nil {
		res := Response{Message: "Could not connect to database"}
		c.Logger().Error(err)
		return c.JSON(http.StatusBadGateway, res)
	}

	cmd := "INSERT INTO items (name, category, image_filename) VALUES (?, ?, ?)"
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

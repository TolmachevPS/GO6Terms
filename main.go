package main

import (
	"bytes"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/davvo/mercator"
	"github.com/fogleman/gg"
	geojson "github.com/paulmach/go.geojson"
)

type Color struct {
	red   float64
	green float64
	blue  float64
}

const WIDTH, HEIGHT = 256, 256
const FILE_PATH = "areas.geojson"

var cache map[string][]byte
var colors []Color

func main() {
	cache = make(map[string][]byte, 0)
	colors = formColors(FILE_PATH)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets/"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/tile/", drawTile)

	http.ListenAndServe(":3000", nil)
}

func formColors(filePath string) []Color {
	var coordinates [][][][][]float64
	var featureCollectionJSON []byte

	featureCollectionJSON, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err.Error())
	}

	coordinates, err = getMultyCoordinates(featureCollectionJSON)
	if err != nil {
		fmt.Println(err.Error())
	}

	colors := make([]Color, len(coordinates))
	for i := 0; i < len(coordinates); i++ {
		colors[i] = Color{red: rand.Float64(), green: rand.Float64(), blue: rand.Float64()}
	}

	return colors
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./index.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	err = t.ExecuteTemplate(w, "index", "hello")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
}

func drawTile(w http.ResponseWriter, r *http.Request) {
	var err error

	key := r.URL.String()
	keys := strings.Split(key, "/")

	z, err := strconv.ParseFloat(keys[2], 64)
	x, err := strconv.ParseFloat(keys[3], 64)
	y, err := strconv.ParseFloat(keys[4], 64)

	var img image.Image
	var imgBytes []byte

	var featureCollectionJSON []byte
	var filePath = FILE_PATH

	if cache[key] != nil {
		imgBytes = cache[key]
	} else {
		if featureCollectionJSON, err = ioutil.ReadFile(filePath); err != nil {
			fmt.Println(err.Error())
		}

		if img, err = getPNG(featureCollectionJSON, z, x, y); err != nil {
			fmt.Println(err.Error())
		}

		buffer := new(bytes.Buffer) //buffer - *bytes.Buffer
		png.Encode(buffer, img)     //img - image.Image
		imgBytes = buffer.Bytes()
		cache[key] = imgBytes
	}

	w.Write(imgBytes)
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func getPNG(featureCollectionJSON []byte, z float64, x float64, y float64) (image.Image, error) {
	var coordinates [][][][][]float64
	var err error

	if coordinates, err = getMultyCoordinates(featureCollectionJSON); err != nil {
		return nil, err
	}

	dc := gg.NewContext(WIDTH, HEIGHT)
	scale := 1.0

	dc.InvertY()
	//рисуем полигоны
	forEachPolygon(dc, coordinates, func(polygonCoordinates [][]float64) {
		drawByPolygonCoordinates(dc, polygonCoordinates, scale, dc.Fill, z, x, y)
	})
	//рисуем контуры полигонов
	dc.SetLineWidth(2)
	forEachPolygon(dc, coordinates, func(polygonCoordinates [][]float64) {
		dc.SetRGB(0.0, 0.0, 0.0)
		drawByPolygonCoordinates(dc, polygonCoordinates, scale, dc.Stroke, z, x, y)
	})

	out := dc.Image()

	return out, nil
}

func getMultyCoordinates(featureCollectionJSON []byte) ([][][][][]float64, error) {
	var featureCollection *geojson.FeatureCollection
	var err error

	if featureCollection, err = geojson.UnmarshalFeatureCollection(featureCollectionJSON); err != nil {
		return nil, err
	}
	var features = featureCollection.Features
	var coordinates [][][][][]float64
	for i := 0; i < len(features); i++ {
		coordinates = append(coordinates, features[i].Geometry.MultiPolygon)
	}
	return coordinates, nil
}

func forEachPolygon(dc *gg.Context, coordinates [][][][][]float64, callback func([][]float64)) {
	for i := 0; i < len(coordinates); i++ {
		dc.SetRGB(colors[i].red, colors[i].green, colors[i].blue)
		for j := 0; j < len(coordinates[i]); j++ {
			callback(coordinates[i][j][0])
		}
	}
}

const mercatorMaxValue float64 = 20037508.342789244

const mercatorToCanvasScaleFactorX = float64(WIDTH) / (mercatorMaxValue)
const mercatorToCanvasScaleFactorY = float64(HEIGHT) / (mercatorMaxValue)

func drawByPolygonCoordinates(dc *gg.Context, coordinates [][]float64, scale float64, method func(), z float64, xTile float64, yTile float64) {

	scale = scale * math.Pow(2, z)

	dx := float64(dc.Width())*(xTile) - 138.5*scale
	dy := float64(dc.Height())*(math.Pow(2, z)-1-yTile) - 128*scale

	for index := 0; index < len(coordinates)-1; index++ {
		x, y := mercator.LatLonToMeters(coordinates[index][1], TransformNegXToPos(coordinates[index][0]))

		x, y = ScaleOnRussia(x, y)

		x *= mercatorToCanvasScaleFactorX * scale * 0.5
		y *= mercatorToCanvasScaleFactorY * scale * 0.5

		x -= dx
		y -= dy

		dc.LineTo(x, y)
	}
	dc.ClosePath()
	method()
}

//ScaleOnRussia is a method to center map coordinates around Russia
func ScaleOnRussia(x float64, y float64) (float64, float64) {
	var west = float64(1635093.15883866)

	if x > 0 {
		x -= west
	} else {
		x += 2*mercatorMaxValue - west
	}

	return x, y
}

//TransformNegXToPos is a method to transform negative x value to positive one
func TransformNegXToPos(x float64) float64 {
	if x < 0 {
		x = x - 360
	}
	return x
}

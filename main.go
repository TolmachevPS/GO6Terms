package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"

	"github.com/fogleman/gg"

	geojson "github.com/paulmach/go.geojson"
)

func main() {
	var fileDest = flag.String("d", "countries.geo.json", "Destination of file")
	flag.Parse()
	fmt.Println(*fileDest)

	dc := gg.NewContext(1366, 1024)

	dc.InvertY()
	dc.Scale(0.00003415, 0.00003415)

	rawFeatureJSON, err := ioutil.ReadFile(*fileDest)
	if err != nil {
		panic(err)
	}

	fc, err := geojson.UnmarshalFeatureCollection(rawFeatureJSON)
	if err != nil {
		panic(err)
	}

	err = drawMap(fc, dc)
	if err != nil {
		panic(err)
	}
	dc.SavePNG("out.png")

}

func drawMap(fc *geojson.FeatureCollection, c *gg.Context) error {
	drawBackground("000", c)
	for _, feature := range fc.Features {
		name, err := feature.PropertyString("name")
		if err != nil {
			return err
		}
		fmt.Println(name)
		drawGeometry(feature.Geometry, c)
	}
	return nil
}

func drawGeometry(g *geojson.Geometry, c *gg.Context) {
	if g.IsCollection() {
		for _, geometry := range g.Geometries {
			drawGeometry(geometry, c)
		}
		return
	}
	if g.IsLineString() {
		drawLine(g.LineString, c)
		return
	}
	if g.IsMultiLineString() {
		for _, line := range g.MultiLineString {
			drawLine(line, c)
		}
		return
	}
	if g.IsMultiPoint() {
		for _, point := range g.MultiPoint {
			drawPoint(point, c)
		}
		return
	}
	if g.IsMultiPolygon() {
		for _, multiPolygon := range g.MultiPolygon {
			for _, polygon := range multiPolygon {
				drawPolygon(polygon, c)
			}
		}
		return
	}
	if g.IsPoint() {
		drawPoint(g.Point, c)
		return
	}
	if g.IsPolygon() {
		for _, polygon := range g.Polygon {
			drawPolygon(polygon, c)
		}
		return
	}
}

func drawLine(l [][]float64, c *gg.Context) {
	x, y := getMercator(l[0][1], l[0][0])
	x += 20000000
	y += 15000000
	c.SetHexColor("0f0")
	c.MoveTo(x, y)
	for _, p := range l {
		x, y = getMercator(p[1], p[0])
		x += 20000000
		y += 15000000
		c.LineTo(p[0], p[1])
	}
}

func drawPoint(p []float64, c *gg.Context) {
	x, y := getMercator(p[1], p[0])
	x += 20000000
	y += 15000000
	c.SetHexColor("00f")
	c.DrawPoint(x, y, 5)
}

func drawPolygon(polygon [][]float64, c *gg.Context) {
	c.SetRGB(0, 0, 0)
	c.SetLineWidth(1.0)
	c.MoveTo(polygon[0][0], polygon[0][1])
	var x float64
	var y float64
	for i := 0; i < len(polygon); i++ {
		x, y = getMercator(polygon[i][1], polygon[i][0])
		x += 20000000
		y += 15000000
		c.LineTo(x, y)
	}
	c.SetHexColor("fff")
	c.Fill()
}

func drawBackground(color string, c *gg.Context) {
	c.MoveTo(0, 0)
	c.LineTo(40000000, 0)
	c.LineTo(40000000, 30000000)
	c.LineTo(0, 30000000)
	c.SetHexColor(color)
	c.Fill()
}

func getMercator(lat float64, long float64) (float64, float64) {
	rLat := lat*math.Pi/180
	rLong := long*math.Pi/180
	a := 6378137.0
	b := 6356752.3142
	f :=(a-b)/a
	e :=math.Sqrt(2*f-math.Pow(f, 2))
	X := a*rLong
	Y := a*math.Log(math.Tan(math.Pi/4+rLat/2)*math.Pow(((1-e*math.Sin(rLat))/(1+e*math.Sin(rLat))), (e/2)))
	return X, Y
}

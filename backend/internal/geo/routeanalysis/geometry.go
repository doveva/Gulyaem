package routeanalysis

import (
	"encoding/json"
	"math"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

const earthRadiusMeters = 6371008.8

type routeSample struct {
	point   domain.Point
	measure float64
	heading float64
}

type nearestPoint struct {
	point    domain.Point
	distance float64
	heading  float64
	measure  float64
}

func distanceMeters(a, b domain.Point) float64 {
	lat := (a.Lat + b.Lat) * math.Pi / 360
	x := (b.Lon - a.Lon) * math.Pi / 180 * earthRadiusMeters * math.Cos(lat)
	y := (b.Lat - a.Lat) * math.Pi / 180 * earthRadiusMeters
	return math.Hypot(x, y)
}

func routeLength(points []domain.Point) float64 {
	total := 0.0
	for index := 1; index < len(points); index++ {
		total += distanceMeters(points[index-1], points[index])
	}
	return total
}

func densify(points []domain.Point, step float64) []routeSample {
	total := routeLength(points)
	if total == 0 {
		return nil
	}
	measures := make([]float64, 0, int(total/step)+2)
	for measure := 0.0; measure < total; measure += step {
		measures = append(measures, measure)
	}
	measures = append(measures, total)
	result := make([]routeSample, 0, len(measures))
	leg, legStart := 0, 0.0
	for _, measure := range measures {
		for leg < len(points)-2 {
			length := distanceMeters(points[leg], points[leg+1])
			if measure <= legStart+length {
				break
			}
			legStart += length
			leg++
		}
		length := distanceMeters(points[leg], points[leg+1])
		ratio := 0.0
		if length > 0 {
			ratio = math.Max(0, math.Min(1, (measure-legStart)/length))
		}
		result = append(result, routeSample{
			point:   interpolate(points[leg], points[leg+1], ratio),
			measure: measure,
			heading: heading(points[leg], points[leg+1]),
		})
	}
	return result
}

func nearestOnLine(point domain.Point, line []domain.Point) nearestPoint {
	best := nearestPoint{distance: math.Inf(1)}
	measure := 0.0
	for index := 1; index < len(line); index++ {
		a, b := line[index-1], line[index]
		length := distanceMeters(a, b)
		if length == 0 {
			continue
		}
		latitude := (a.Lat + b.Lat + point.Lat) / 3 * math.Pi / 180
		scaleX := earthRadiusMeters * math.Pi / 180 * math.Cos(latitude)
		scaleY := earthRadiusMeters * math.Pi / 180
		bx, by := (b.Lon-a.Lon)*scaleX, (b.Lat-a.Lat)*scaleY
		px, py := (point.Lon-a.Lon)*scaleX, (point.Lat-a.Lat)*scaleY
		ratio := math.Max(0, math.Min(1, (px*bx+py*by)/(bx*bx+by*by)))
		projected := interpolate(a, b, ratio)
		distance := distanceMeters(point, projected)
		if distance < best.distance {
			best = nearestPoint{point: projected, distance: distance, heading: heading(a, b), measure: measure + length*ratio}
		}
		measure += length
	}
	return best
}

func interpolate(a, b domain.Point, ratio float64) domain.Point {
	return domain.Point{Lon: a.Lon + (b.Lon-a.Lon)*ratio, Lat: a.Lat + (b.Lat-a.Lat)*ratio}
}

func heading(a, b domain.Point) float64 {
	latitude := (a.Lat + b.Lat) * math.Pi / 360
	x := (b.Lon - a.Lon) * math.Cos(latitude)
	y := b.Lat - a.Lat
	angle := math.Atan2(x, y) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	return angle
}

func undirectedAngleDifference(a, b float64) float64 {
	difference := math.Abs(a - b)
	if difference > 180 {
		difference = 360 - difference
	}
	if difference > 90 {
		difference = 180 - difference
	}
	return difference
}

func lineGeometryJSON(points []domain.Point) json.RawMessage {
	coordinates := make([][2]float64, len(points))
	for index, point := range points {
		coordinates[index] = [2]float64{point.Lon, point.Lat}
	}
	result, _ := json.Marshal(struct {
		Type        string       `json:"type"`
		Coordinates [][2]float64 `json:"coordinates"`
	}{Type: "LineString", Coordinates: coordinates})
	return result
}

func multiLineGeometryJSON(lines [][]domain.Point) json.RawMessage {
	coordinates := make([][][2]float64, 0, len(lines))
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		converted := make([][2]float64, len(line))
		for index, point := range line {
			converted[index] = [2]float64{point.Lon, point.Lat}
		}
		coordinates = append(coordinates, converted)
	}
	result, _ := json.Marshal(struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}{Type: "MultiLineString", Coordinates: coordinates})
	return result
}

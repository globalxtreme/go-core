package xtrememodel

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// Untuk menentukan dimensi:
// XY: Ini untuk 2D. X: Longitude, Y: Latitude
// XYZ: Ini untuk 3D. X: Longitude, Y: Latitude, Z: Ketinggian
// XYM: Ini untuk 2D namun ada nilai tambahan seperti jawak, waktu, dll. X: Longitude, Y: Latitude, M: Titik awal
// XYZM: Ini untuk 3D namun ada nilai tambahan. X: Longitude, Y: Latitude, Z: Ketinggian, M: Titik awal

var SRID = 4326

type (
	Geometry[T geom.T] struct{ Geom T }

	Point              = Geometry[*geom.Point]
	LineString         = Geometry[*geom.LineString]
	Polygon            = Geometry[*geom.Polygon]
	MultiPoint         = Geometry[*geom.MultiPoint]
	MultiLineString    = Geometry[*geom.MultiLineString]
	MultiPolygon       = Geometry[*geom.MultiPolygon]
	GeometryCollection = Geometry[*geom.GeometryCollection]
)

func NewGeometry[T geom.T](geom T) Geometry[T] { return Geometry[T]{geom} }

func (g *Geometry[T]) Scan(value interface{}) (err error) {
	var (
		wkb []byte
		ok  bool
	)

	switch v := value.(type) {
	case string:
		wkb, err = hex.DecodeString(v)
	case []byte:
		wkb = v
	default:
		return errors.New("unexpected geometry type")
	}

	if err != nil {
		return err
	}

	geometryT, err := ewkb.Unmarshal(wkb)
	if err != nil {
		return err
	}

	g.Geom, ok = geometryT.(T)
	if !ok {
		return errors.New("unexpected value type")
	}

	return
}

func (g Geometry[T]) Value() (driver.Value, error) {
	if geom.T(g.Geom) == nil {
		return nil, nil
	}

	sb := &bytes.Buffer{}
	if err := ewkb.Write(sb, binary.LittleEndian, g.Geom); err != nil {
		return nil, err
	}

	return hex.EncodeToString(sb.Bytes()), nil
}

func (g Geometry[T]) GormDataType() string {
	srid := strconv.Itoa(SRID)

	switch any(g.Geom).(type) {
	case *geom.Point:
		return "Geometry(Point, " + srid + ")"
	case *geom.LineString:
		return "Geometry(LineString, " + srid + ")"
	case *geom.Polygon:
		return "Geometry(Polygon, " + srid + ")"
	case *geom.MultiPoint:
		return "Geometry(MultiPoint, " + srid + ")"
	case *geom.MultiLineString:
		return "Geometry(MultiLineString, " + srid + ")"
	case *geom.MultiPolygon:
		return "Geometry(MultiPolygon, " + srid + ")"
	case *geom.GeometryCollection:
		return "Geometry(GeometryCollection, " + srid + ")"
	default:
		return "geometry"
	}
}

func (g Geometry[T]) String() string {
	if geomWkt, err := wkt.Marshal(g.Geom); err == nil {
		return geomWkt
	}

	return fmt.Sprintf("cannot marshal geometry: %T", g.Geom)
}

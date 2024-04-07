/*
Copyright (c) 2017 Lauris Bukšis-Haberkorns <lauris@nix.lv> and contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package tiled

import (
	"encoding/xml"
	"errors"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrInvalidObjectPoint error is returned if there is error parsing object points
var ErrInvalidObjectPoint = errors.New("tiled: invalid object point")

// ObjectGroup is in fact a map layer, and is hence called "object layer" in Tiled Qt
type ObjectGroup struct {
	// Unique ID of the layer.
	// Each layer that added to a map gets a unique id. Even if a layer is deleted,
	// no layer ever gets the same ID. Can not be changed in Tiled. (since Tiled 1.2)
	ID uint32 `xml:"id,attr"`
	// The name of the object group.
	Name string `xml:"name,attr"`
	// The class of the object group (since 1.9, defaults to "").
	Class string `xml:"class,attr"`
	// The color used to display the objects in this group.
	Color *HexColor `xml:"color,attr"`
	// The opacity of the layer as a value from 0 to 1. Defaults to 1.
	Opacity float32 `xml:"opacity,attr"`
	// Whether the layer is shown (1) or hidden (0). Defaults to 1.
	Visible bool `xml:"visible,attr"`
	// Rendering offset for this layer in pixels. Defaults to 0. (since 0.14)
	OffsetX int `xml:"offsetx,attr"`
	// Rendering offset for this layer in pixels. Defaults to 0. (since 0.14)
	OffsetY int `xml:"offsety,attr"`
	// Whether the objects are drawn according to the order of appearance ("index") or sorted by their y-coordinate ("topdown"). Defaults to "topdown".
	DrawOrder string `xml:"draworder,attr"`
	// The parallax x factor of the layer 0 - 1.0
	ParallaxX float32 `xml:"parallaxx,attr"`
	// The parallax y factor of the layer 0 - 1.0
	ParallaxY float32 `xml:"parallaxy,attr"`
	// Custom properties
	Properties Properties `xml:"properties>property"`
	// Group objects
	Objects []*Object `xml:"object"`
}

// DecodeObjectGroup decodes object group data
func (g *ObjectGroup) DecodeObjectGroup(m *Map) error {
	for _, object := range g.Objects {
		if object.GID > 0 {
			// Initialize all tilesets that are referenced by tile objects. Otherwise,
			// if a tileset is used by an object tile but not used by any layer it
			// won't be loaded.
			if _, err := m.TileGIDToTile(object.GID); err != nil {
				return err
			}
		}
		if len(object.TemplateSource) > 0 {
			if err := object.initTemplate(m); err != nil {
				return err
			}
		}
	}
	return nil
}

// UnmarshalXML decodes a single XML element beginning with the given start element.
func (g *ObjectGroup) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	item := aliasObjectGroup{}
	item.SetDefaults()

	if err := d.DecodeElement(&item, &start); err != nil {
		return err
	}

	*g = (ObjectGroup)(item)

	return nil
}

// Object is used to add custom information to your tile map, such as spawn points, warps, exits, etc.
type Object struct {
	// Unique ID of the object. Each object that is placed on a map gets a unique id. Even if an object was deleted, no object gets the same ID.
	// Can not be changed in Tiled Qt. (since Tiled 0.11)
	ID uint32 `xml:"id,attr"`
	// The name of the object. An arbitrary string.
	Name string `xml:"name,attr"`
	// The type of the object. An arbitrary string. (until 1.8)
	//
	// Deprecated: replaced by Class since 1.9
	Type string `xml:"type,attr"`
	// The class of the object. An arbitrary string. (defaults to "", renamed from 'type' since 1.9)
	Class string `xml:"class,attr"`
	// The x coordinate of the object.
	X float64 `xml:"x,attr"`
	// The y coordinate of the object.
	Y float64 `xml:"y,attr"`
	// The width of the object (defaults to 0).
	Width float64 `xml:"width,attr"`
	// The height of the object (defaults to 0).
	Height float64 `xml:"height,attr"`
	// The rotation of the object in degrees clockwise (defaults to 0). (since 0.10)
	Rotation float64 `xml:"rotation,attr"`
	// An reference to a tile (optional).
	GID uint32 `xml:"gid,attr"`
	// Whether the object is shown (1) or hidden (0). Defaults to 1. (since 0.9)
	Visible bool `xml:"visible,attr"`
	// Custom properties
	Properties Properties `xml:"properties>property"`
	// Used to mark an object as an ellipse. The existing x, y, width and height attributes are used to determine the size of the ellipse.
	Ellipses []*Ellipse `xml:"ellipse"`
	// Polygons
	Polygons []*Polygon `xml:"polygon"`
	// Poly lines
	PolyLines []*PolyLine `xml:"polyline"`
	// Text
	Text *Text `xml:"text"`
	// Template
	TemplateSource string `xml:"template,attr"`
	TemplateLoaded bool   `xml:"-"`
	Template       *Template
}

func (o *Object) initTemplate(m *Map) error {
	if o.TemplateLoaded {
		return nil
	}
	if len(o.TemplateSource) == 0 {
		o.TemplateLoaded = true
		return nil
	}
	sourcePath := m.GetFileFullPath(o.TemplateSource)
	f, err := m.loader.open(sourcePath)
	if err != nil {
		return err
	}
	defer f.Close()

	d := xml.NewDecoder(f)
	if err := d.Decode(&o.Template); err != nil {
		return err
	}
	o.TemplateLoaded = true

	if o.Template == nil || o.Template.Tileset == nil || o.Template.Object == nil {
		return nil
	}
	if src := o.Template.Tileset.Source; len(src) > 0 {
		// The tileset source may be relative from the template location.
		o.Template.Tileset.Source = path.Join(filepath.Dir(o.TemplateSource), src)
	}
	return m.initTileset(o.Template.Tileset)
}

// UnmarshalXML decodes a single XML element beginning with the given start element.
func (o *Object) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	item := aliasObject{}
	item.SetDefaults()

	if err := d.DecodeElement(&item, &start); err != nil {
		return err
	}

	*o = (Object)(item)

	return nil
}

// Ellipse is used to mark an object as an ellipse.
type Ellipse struct{}

// Polygon object is made up of a space-delimited list of x,y coordinates. The origin for these coordinates is the location of the parent object.
// By default, the first point is created as 0,0 denoting that the point will originate exactly where the object is placed.
type Polygon struct {
	// A list of x,y coordinates in pixels.
	Points *Points `xml:"points,attr"`
}

// PolyLine follows the same placement definition as a polygon object.
type PolyLine struct {
	// A list of x,y coordinates in pixels.
	Points *Points `xml:"points,attr"`
}

// Point is point
type Point struct {
	// Point X
	X float64
	// Point Y
	Y float64
}

// Points is array of points
type Points []*Point

// UnmarshalXMLAttr decodes a single XML element beginning with the given start element.
func (m *Points) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value == "" {
		return nil
	}

	ps := strings.Split(attr.Value, " ")

	points := make(Points, len(ps))

	for i, s := range ps {
		c := strings.Split(s, ",")
		if len(c) != 2 {
			return ErrInvalidObjectPoint
		}

		var x, y float64
		var err error
		if x, err = strconv.ParseFloat(c[0], 64); err != nil {
			return err
		}
		if y, err = strconv.ParseFloat(c[1], 64); err != nil {
			return err
		}
		points[i] = &Point{
			X: x,
			Y: y,
		}
	}

	*m = points
	return nil
}

// Text contains a text and attributes such as bold, color, etc.
type Text struct {
	// The actual text
	Text string `xml:",chardata"`
	// The font family used (default: "sans-serif")
	FontFamily string `xml:"fontfamily,attr"`
	// The size of the font in pixels (not using points, because other sizes in the TMX format are also using pixels) (default: 16)
	Size int `xml:"pixelsize,attr"`
	// Whether word wrapping is enabled (1) or disabled (0). Defaults to 0.
	Wrap bool `xml:"wrap,attr"`
	// Color of the text in #AARRGGBB or #RRGGBB format (default: #000000)
	Color *HexColor `xml:"color,attr"`
	// Whether the font is bold (1) or not (0). Defaults to 0.
	Bold bool `xml:"bold,attr"`
	// Whether the font is italic (1) or not (0). Defaults to 0.
	Italic bool `xml:"italic,attr"`
	// Whether a line should be drawn below the text (1) or not (0). Defaults to 0.
	Underline bool `xml:"underline,attr"`
	// Whether a line should be drawn through the text (1) or not (0). Defaults to 0.
	Strikethrough bool `xml:"strikeout,attr"`
	// Whether kerning should be used while rendering the text (1) or not (0). Default to 1.
	Kerning bool `xml:"kerning,attr"`
	// Horizontal alignment of the text within the object (left (default), center, right or justify (since Tiled 1.2.1))
	HAlign string `xml:"halign,attr"`
	// Vertical alignment of the text within the object (top (default), center or bottom)
	VAlign string `xml:"valign,attr"`
}

// UnmarshalXML decodes a single XML element beginning with the given start element.
func (t *Text) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	item := aliasText{}
	item.SetDefaults()

	if err := d.DecodeElement(&item, &start); err != nil {
		return err
	}

	*t = (Text)(item)

	return nil
}

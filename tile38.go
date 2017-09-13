package tile38

type Tile38Client interface {
	Do(string, ...interface{}) (interface{}, error)
	Endpoint() string
}

type Tile38Coord struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type Tile38Point struct {
	ID     string        `json:"id"`
	Point  Tile38Coord   `json:"point"`
	Fields []interface{} `json:"fields"`
}

type Tile38Response struct {
	Ok     bool          `json:"ok"`
	Count  int           `json:"count,omitempty"`
	Cursor int           `json:"cursor,omitempty"`
	Fields interface{}   `json:"fields,omitempty"`
	Points []Tile38Point `json:"points,omitempty"`
	Object interface{}   `json:"object,omitempty"`
}

// 2017/04/19 20:50:53 {"ok":true,"object":"{\"wof:name\":\"10128\",\"wof:country\":\"US\"}","elapsed":"28.496µs"}
// 2017/04/19 20:50:53 {"ok":false,"err":"id not found","elapsed":"18.15µs"}

type Tile38MetaResponse struct {
	Ok      bool   `json:"ok"`
	Elapsed string `json:"elapsed,omitempty"`
	Error   string `json:"err,omitempty"`
	Object  string `json:"object,omitempty"`
}

/*
	perhaps you're wondering what the relationship is between these and
	https://github.com/whosonfirst/go-whosonfirst-api and the answer
	is so am I, so am I... (20170305/thisisaaronland)

	translation: please update this to use go-whosonfirst-spr
	(20170801/thisisaaronland)
*/

type WOFResponse struct {
	Results []WOFResult `json:"results"`
	Cursor  int         `json:"cursor"`
}

type WOFResult struct {
	WOFID          int64   `json:"wof:id"`
	WOFParentID    int64   `json:"wof:parent_id"`
	WOFPlacetypeID int64   `json:"wof:placetype_id"`
	WOFSuperseded  int64   `json:"wof:is_superseded"`
	WOFDeprecated  int64   `json:"wof:is_deprecated"`
	WOFRepo        string  `json:"wof:repo"`
	WOFName        string  `json:"wof:name"`
	WOFCountry     string  `json:"wof:country"`
	GeomLatitude   float64 `json:"geom:latitude"`
	GeomLongitude  float64 `json:"geom:longitude"`
}

type WOFMetaResult struct {
	WOFName    string `json:"wof:name"`
	WOFCountry string `json:"wof:country"`
}

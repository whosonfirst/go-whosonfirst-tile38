package tile38

type Tile38Coord struct {
     Latitude float64 `json:"lat"`
     Longitude float64 `json:"lon"`     
}

type Tile38Point struct {
     ID string `json:"id"`
     Point Tile38Coord `json:"point"`
     Fields []interface{} `json:"fields"`
}

type Tile38Response struct {
     Ok bool `json:"ok"`
     Count int `json:"count"`
     Cursor int `json:"cursor"`
     Fields []string `json:"fields"`
     Points []Tile38Point `json:"points"`
}

type WOFResponse struct {
     Results []WOFResult `json:"results"`
     Cursor  int `json:"cursor"`     
}

type WOFResult struct {
     WOFID     int64 `json:"wof:id"`
     WOFParentID     int64 `json:"wof:parent_id"`
     WOFPlacetypeID     int64 `json:"wof:placetype_id"`
     WOFSuperseded     int64 `json:"wof:is_superseded"`
     WOFDeprecated     int64 `json:"wof:is_deprecated"`
}

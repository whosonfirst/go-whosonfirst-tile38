package whosonfirst

import (
	"encoding/json"
	"errors"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"strings"
)

func Tile38ResponseToWOFResponse(rsp tile38.Tile38Response) (tile38.WOFResponse, error) {

	wof_results := make([]tile38.WOFResult, 0)

	for _, p := range rsp.Points {

		pt := p.Point

		parts := strings.Split(p.ID, "#")
		repo := parts[1]

		tmp := make(map[string]int64)

		// sometimes 'fields' is a list of strings, sometimes it's a list of ints
		// and sometimes... it's a dictionary (20170913/thisisaaronland)

		fields := rsp.Fields.([]interface{})

		for i, k := range fields {
			str_k := k.(string)
			v := int64(p.Fields[i].(float64))
			tmp[str_k] = v
		}

		wof_result := tile38.WOFResult{
			WOFID:          tmp["wof:id"],
			WOFParentID:    tmp["wof:parent_id"],
			WOFPlacetypeID: tmp["wof:placetype_id"],
			WOFSuperseded:  tmp["wof:is_superseded"],
			WOFDeprecated:  tmp["wof:is_deprecated"],
			WOFRepo:        repo,
			GeomLatitude:   pt.Latitude,
			GeomLongitude:  pt.Longitude,
		}

		wof_results = append(wof_results, wof_result)
	}

	wof_response := tile38.WOFResponse{
		Cursor:  rsp.Cursor,
		Results: wof_results,
	}

	return wof_response, nil
}

func Tile38MetaResponseToWOFMetaResult(rsp tile38.Tile38MetaResponse) (*tile38.WOFMetaResult, error) {

	if !rsp.Ok {

		err := errors.New("Tile38MetaResponse was not successful")
		return nil, err
	}

	var wof_rsp tile38.WOFMetaResult

	err := json.Unmarshal([]byte(rsp.Object), &wof_rsp)

	if err != nil {
		return nil, err
	}

	return &wof_rsp, nil
}

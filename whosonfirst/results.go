package whosonfirst

import (
	"github.com/whosonfirst/go-whosonfirst-tile38"
)

func Tile38ResponseToWOFResponse(rsp tile38.Tile38Response) (tile38.WOFResponse, error) {

	wof_results := make([]tile38.WOFResult, 0)

	for _, p := range rsp.Points {

		tmp := make(map[string]int64)

		for i, k := range rsp.Fields {
			v := int64(p.Fields[i].(float64))
			tmp[k] = v
		}

		wof_result := tile38.WOFResult{
			WOFID:          tmp["wof:id"],
			WOFParentID:    tmp["wof:parent_id"],
			WOFPlacetypeID: tmp["wof:placetype_id"],
			WOFSuperseded:  tmp["wof:is_superseded"],
			WOFDeprecated:  tmp["wof:is_deprecated"],
		}

		wof_results = append(wof_results, wof_result)
	}

	wof_response := tile38.WOFResponse{
		Cursor:  rsp.Cursor,
		Results: wof_results,
	}

	return wof_response, nil
}

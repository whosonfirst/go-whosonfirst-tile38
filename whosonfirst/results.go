package whosonfirst

import (
	"github.com/whosonfirst/go-whosonfirst-tile38/whosonfirst"
)

func Tile38ResponseToWOFResults(rsp tile38.Tile38Response) (tile38.WOFResults, error) {

	wof_results := make([]tile38.WOFResult, 0)

	for _, p := range r.Points {

		tmp := make(map[string]int64)

		for i, k := range r.Fields {
			v := int64(p.Fields[i].(float64))
			tmp[k] = v
		}

		wof_result := WOFResult{
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

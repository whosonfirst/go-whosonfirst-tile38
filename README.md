# go-whosonfirst-tile38

Go tools for working with Who's On First documents and [Tile38](http://tile38.com)

## Caveat

This is tilting towards ready on the experimental-to-ready scale, but isn't quite there yet.

## Install

You will need to have both `Go` and the `make` programs installed on your computer. Assuming you do just type:

```
make bin
```

All of this package's dependencies are bundled with the code in the `vendor` directory.

## Indexing

Indexing Who's On First data is done using the `wof-tile38-index` utility described below. There are a few important things to remember about indexing:

1. This is very much Who's On First (Mapzen) specific
2. The goal is to index _as little_ extra information as possible which ensuring the ability to generate a "minimal viable WOF record" and to perform basic container-ish (belongs to, placetype, etc.) queries.
3. Those details are still a moving target.
4. To whit, see the way we're encoding the repository name in to the first key itself? That is perhaps unnecessary.
5. Whatever else this package holds hands with, now or in the future, it currently holds hands (tightly) with the `lib_whosonfirst_spatial.php` library in the [whosonfirst-www-api](https://github.com/whosonfirst/whosonfirst-www-api) repo and the [py-mapzen-whosonfirst-tile38](https://github.com/whosonfirst/py-mapzen-whosonfirst-tile38) library.

### _WOFID_ + "#" + _REPO_NAME_

This stores a record's geometry (which may be a centroid or the actual GeoJSON `geometry` property) as well as the following numeric IDs:

* wof:id
* wof:placetype_id
* wof:parent_id
* wof:is_deprecated	(stored as `0` or `1`)
* wof:is_superseded	(stored as `0` or `1`)

This we probably _should_ store but aren't yet:

* mz:is_current
* mz:scale_rank

### _WOFID_ + "#meta"

This stores a following fields as a JSON encoded dictionary:

* wof:name
* wof:country

## Querying

Querying Who's On First data can either be done by talking to a Tile38 server directly or using one of the utilities described below (currently there is only one).

### Query results

Unless otherwise noted all query results are return as a simple list of JSON dictionaries. Paginated is implemented using a `cursor` parameter which is returned at top level of any query response. If present you should include it with the following query to return the next set of results. For example:

```
$> curl -s 'localhost:8080?bbox=-33.893217,151.165524,-33.840479,151.281223&per_page=1' | python -mjson.tool
{
    "cursor": 1,
    "results": [
        {
            "wof:id": 1108814711,
            "wof:is_deprecated": 0,
            "wof:is_superseded": 0,
            "wof:parent_id": -1,
            "wof:placetype_id": 102312319,
            "wof:repo": "dxlabs"
        }
    ]
}
```

## Utilities

_All of these utilities assume that there is a running copy of the `tile38-server` (included in [Tile38](https://github.com/tidwall/tile38/) package) that these utilities can communicate with. This package defines tools and utilities for working with `tile38-server` not to replace it._

### wof-tile38-bboxd

Find Who's On First records in a Tile38 database that intersect a given bounding box. 

```
./bin/wof-tile38-bboxd -h
Usage of ./bin/wof-tile38-bboxd:
  -host string
    	The address your HTTP server should listen for requests on (default "localhost")
  -port int
    	The port number your HTTP server should listen for requests on (default 8080)
  -tile38-collection string
    	The name of the Tile38 collection to read data from.
  -tile38-host string
    	The address your Tile38 server is bound to. (default "localhost")
  -tile38-port int
    	The port number your Tile38 server is bound to. (default 9851)
```

#### Query parameters

* **bbox** _required_ – Any bounding box format that is supported by the [go-whosonfirst-bbox](https://github.com/whosonfirst/go-whosonfirst-bbox) package.
* ** scheme** _optional_ A valid [go-whosonfirst-bbox scheme](https://github.com/whosonfirst/go-whosonfirst-bbox#schemes)
* ** scheme** _optional_ A valid [go-whosonfirst-bbox order](https://github.com/whosonfirst/go-whosonfirst-bbox#order)
* **per_page** _optional_ The number of results to include with a query. Default is 100.
* **cursor** _optional_ A pointer to the next set of results for your query.

#### Example

_This assumes you've created an index called `dxlabs`. See below for details._

```
$> wof-tile38-bboxd -tile38-collection dxlabs

$> curl 'localhost:8080?bbox=-33.893217,151.165524,-33.840479,151.281223&per_page=1&cursor=1'
{"results":[{"wof:id":1108823025,"wof:parent_id":-1,"wof:placetype_id":102312319,"wof:is_superseded":0,"wof:is_deprecated":0}],"cursor":2}
```

### wof-tile38-index

Index one or more Who's On First records in a Tile38 database.

```
./bin/wof-tile38-index -h
Usage of ./bin/wof-tile38-index:
  -debug
    	Go through all the motions but don't actually index anything.
  -geometry string
    	Which geometry to index. Valid options are: centroid, bbox or whatever is in the default GeoJSON geometry (default).
  -mode string
    	The mode to use importing data. Valid options are: directory, filelist and files. (default "files")
  -nfs-kludge
    	Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors
  -procs int
    	The number of concurrent processes to use importing data. (default is number of CPUs * 2)
  -tile38-collection string
    	The name of the Tile38 collection for indexing data.
  -tile38-host string
    	The address your Tile38 server is bound to. (default "localhost")
  -tile38-port int
    	The port number your Tile38 server is bound to. (default 9851)
  -verbose
    	Be chatty about what's happening. This is automatically enabled if the -debug flag is set.
```

#### Example

For example, if you wanted to index [all the localities](https://whosonfirst.mapzen.com/bundles/#placetypes-common) in Who's On First:

```
$> wget https://whosonfirst.mapzen.com/bundles/wof-locality-latest-bundle.tar.bz2
$> tar -xvjf wof-locality-latest-bundle.tar.bz2
$> wof-tile38-index -collection whosonfirst -mode directory wof-locality-latest-bundle/data/
```

If you wanted to index one or more Who's On First "meta" files (they're just CSV files with a `path` column) you might do something like:

```
$> wof-tile38-index -tile38-collection whosonfirst-geom -mode meta /usr/local/data/whosonfirst-data/meta/wof-county-latest.csv:/usr/local/data/whosonfirst-data/data
```

The syntax for listing meta files to index is a pair of filesystem paths separated by a `:`. The first path is the path to the meta file and the second is the path to the directory containing the actual GeoJSON files. As of this writing it is assumed that the paths listed in the meta files are relative.

## See also

* http://tile38.com
* https://github.com/whosonfirst/go-whosonfirst-geojson-v2
* https://github.com/whosonfirst/go-whosonfirst-bbox
* https://github.com/whosonfirst/go-whosonfirst-crawl

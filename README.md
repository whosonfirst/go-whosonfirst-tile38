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

## Utilities

### wof-tile38-index

```
./bin/wof-tile38-index -h
Usage of ./bin/wof-tile38-index:
  -collection string
    	The name of the Tile38 collection for indexing data.
  -debug
    	Go through all the motions but don't actually index anything.
  -geometry string
    	Which geometry to index. Valid options are: centroid, bbox or whatever is in the default GeoJSON geometry ("").
  -mode string
    	The mode to use importing data. Valid options are: directory, filelist and files. (default "files")
  -nfs-kludge
    	Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors
  -procs int
    	The number of concurrent processes to use importing data. (default 200)
  -tile38-host string
    	The host of your Tile-38 server. (default "localhost")
  -tile38-port int
    	The port of your Tile38 server. (default 9851)
```

#### Example

For example, if you wanted to index [all the localities](https://whosonfirst.mapzen.com/bundles/#placetypes-common) in Who's On First:

```
$> wget https://whosonfirst.mapzen.com/bundles/wof-locality-latest-bundle.tar.bz2
$> tar -xvjf wof-locality-latest-bundle.tar.bz2
$> wof-tile38-index -procs 200 -collection whosonfirst-geom -procs 200 -mode directory wof-locality-latest-bundle/data/
```

If you wanted to index one or more Who's On First "meta" files (they're just CSV files with a `path` column) you might do something like:

```
$> wof-tile38-index -collection whosonfirst-geom -mode meta /usr/local/data/whosonfirst-data/meta/wof-county-latest.csv:/usr/local/data/whosonfirst-data/data
```

The syntax for listing meta files to index is a pair of filesystem paths separated by a `:`. The first path is the path to the meta file and the second is the path to the directory containing the actual GeoJSON files. As of this writing it is assumed that the paths listed in the meta files are relative.

## Indexing

There are a few important things to remember about indexing:

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
* wof:hierarchy

## See also

* http://tile38.com
* https://github.com/whosonfirst/py-mapzen-whosonfirst-tile38
* https://github.com/whosonfirst/go-whosonfirst-crawl
* https://github.com/whosonfirst/go-whosonfirst-geojson

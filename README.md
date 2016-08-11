# go-whosonfirst-tile38

Go tools for working with Who's On First documents and [Tile38]()

## Caveat

This is experimental, still.

## Requirements

Here's what things look like after indexing 15M Who's On First records in a vanilla instance of Tile38. 

```
127.0.0.1:9851> STATS whosonfirst
{"ok":true,"stats":[{"in_memory_size":5624824430,"num_objects":15148756,"num_points":225332409}],"elapsed":"76.19µs"}

127.0.0.1:9851> SERVER
{"ok":true,"stats":{"aof_size":11111797665,"avg_item_size":43,"heap_size":9794365216,"id":"c5ce956e83931f71774a48d2eccfcb19","in_memory_size":5624824430,"max_heap_size":0,"num_collections":1,"num_hooks":0,"num_objects":15148756,"num_points":225332409,"pointer_size":8,"read_only":false},"elapsed":"13.772854ms"}
```

And this:

```
du -h /mnt/data/appendonly.aof 
11G     /mnt/data/appendonly.aof
```

## See also

* http://tile38.com
* https://github.com/whosonfirst/py-mapzen-whosonfirst-tile38
* https://github.com/whosonfirst/go-whosonfirst-crawl
* https://github.com/whosonfirst/go-whosonfirst-geojson

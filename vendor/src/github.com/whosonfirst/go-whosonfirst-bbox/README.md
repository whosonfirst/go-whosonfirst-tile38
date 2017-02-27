# go-whosonfirst-bbox

Too soon. Move along.

## Usage

### Simple

```
package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-bbox/parser"
	"log"
)

func main() {

	var bbox = flag.String("bbox", "", "A valid bounding box")
	flag.Parse()

	p, _ := parser.NewParser()
	bb, _ := p.Parse(*bbox)

	fmt.Print(bb)
}
```

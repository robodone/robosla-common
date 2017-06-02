package main

import (
	"fmt"
	"log"

	"github.com/robodone/robosla-common/pkg/autoupdate"
)

func main() {
	m, err := autoupdate.FetchAndParseManifest(autoupdate.ProdManifestURL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(m)
}

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/robodone/robosla-common/pkg/autoupdate"
)

var version = flag.String("version", "dev", "Version to test with")

func main() {
	flag.Parse()
	fmt.Printf("Version: %s\n", *version)
	fmt.Printf("IsDevBuild: %v\n", autoupdate.IsDevBuild(*version))
	m, err := autoupdate.FetchAndParseManifest(autoupdate.ProdManifestURL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(m)
	fmt.Printf("NeedsUpdate: %v\n", m.NeedsUpdate(*version))
}

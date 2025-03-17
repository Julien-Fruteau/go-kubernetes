package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/julien-fruteau/go-kubernetes/external/k8s"
)

func main() {
	output := flag.String("output", "json", "output format: json or raw")
	flag.Parse()

	k, err := k8s.NewK8SOutCli(context.Background())
	if err != nil {
		panic(err)
	}

	images, err := k.GetClusterImagesV1()
	// images, err := k.GetClusterImagesV2()
	if err != nil {
    fmt.Println(err)
		os.Exit(1)
	}
	slices.Sort(images)

	// fmt.Println(images)
	switch *output {
	case "json":
		jsonData, err := json.Marshal(images)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Fprintln(os.Stdout, string(jsonData))
	case "raw":
		fmt.Println(images)
	}
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/lukasmoellerch/vvz-go/pkg/vvzfetch"
	"github.com/lukasmoellerch/vvz-go/pkg/vvzscrape"
)

func main() {

	ctx := context.Background()
	courses, _, err := vvzfetch.FetchAllCourses(ctx, "2022W", "en")
	if err != nil {
		log.Fatal(err)
	}

	jsonFile, err := os.Create("./data/courses.json")
	if err != nil {
		log.Fatalln(err)
	}
	defer jsonFile.Close()

	courseArray := make([]*vvzscrape.Course, 0)
	for _, course := range courses {
		courseArray = append(courseArray, course)
	}
	if err = json.NewEncoder(jsonFile).Encode(courseArray); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Wrote %d courses to data/courses.json.\n", len(courseArray))
}

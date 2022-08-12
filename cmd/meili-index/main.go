package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/lukasmoellerch/vvz-go/pkg/vvzfetch"
	"github.com/lukasmoellerch/vvz-go/pkg/vvzscrape"
	"github.com/meilisearch/meilisearch-go"
)

var nonAsciiRegex = regexp.MustCompile(`[^\x00-\x7F]`)

func getSearchableAttributes() *[]string {
	return &[]string{
		"title",
		"lecturers",
		"readableId",
		"abstract",
		"notice",
		"content",
	}
}

func getFilterableAttributes() *[]string {
	return &[]string{
		"lecturers",
		"courseType",
		"ects",
		"level0",
		"level1",
		"level2",
		"level3",
		"level4",
		"level5",
		"level6",
	}
}

func cleanString(str string) string {
	return nonAsciiRegex.ReplaceAllString(str, "")
}

func insertIntoIndex(indexName string, courses map[string]*vvzscrape.Course) error {
	fmt.Printf("Found %d courses.\n", len(courses))

	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   "http://127.0.0.1:7700",
		APIKey: "MASTER_KEY",
	})

	index := client.Index("courses")
	deleteAllUpdate, err := index.DeleteAllDocuments()
	if err != nil {
		fmt.Printf("Delete failed, probably because index doesn't exist\n")
	} else {
		index.WaitForTask(deleteAllUpdate.TaskUID)
	}

	updateSearchUpdate, err := index.UpdateSearchableAttributes(getSearchableAttributes())
	if err != nil {
		panic(err)
	}
	index.WaitForTask(updateSearchUpdate.TaskUID)

	updateFilterUpdate, err := index.UpdateFilterableAttributes(getFilterableAttributes())
	if err != nil {
		panic(err)
	}
	index.WaitForTask(updateFilterUpdate.TaskUID)

	updateRankingUpdate, err := index.UpdateRankingRules(&[]string{
		"words",
		"attribute",
		"typo",
		"proximity",
		"sort",
		"exactness",
	})
	if err != nil {
		panic(err)
	}
	index.WaitForTask(updateRankingUpdate.TaskUID)

	fmt.Printf("Deleted all documents.\n")

	chunkSize := 128
	currentChunk := []map[string]interface{}{}
	updates := make([]int64, 0)
	for _, course := range courses {
		lecturerDocs := make([]map[string]interface{}, len(course.Lecturer))

		for i, lecturer := range course.Lecturer {
			lecturerDocs[i] = map[string]interface{}{
				"id":         lecturer.Id,
				"department": lecturer.Department,
				"firstName":  lecturer.FirstName,
				"lastName":   lecturer.LastName,
				"title":      lecturer.Title,
			}
		}

		doc := map[string]interface{}{
			"id":         strconv.Itoa(course.Id),
			"readableId": course.ReadableId,
			"segments":   course.Segments,
			"title":      course.Title,
			"courseType": course.CourseType,
			"ects":       course.Ects,
			"hours":      course.Hours,
			"lecturers":  lecturerDocs,
			"abstract":   cleanString(course.Abstract),
			"objective":  cleanString(course.Objective),
			"content":    cleanString(course.Content),
			"notice":     cleanString(course.Notice),
		}

		i := 1
		for {
			level := make([]string, 0)
			for _, segment := range course.Segments {
				if len(segment) < i {
					continue
				}
				s := ""
				for _, segmentPart := range segment[:i] {
					if s != "" {
						s += " > "
					}
					s += segmentPart
				}

				contained := false
				for _, levelPart := range level {
					if levelPart == s {
						contained = true
						break
					}
				}

				if !contained {
					level = append(level, s)
				}
			}
			if len(level) > 0 {
				doc["level"+strconv.Itoa(i)] = level
			} else {
				break
			}

			i++
		}

		currentChunk = append(currentChunk, doc)
		if len(currentChunk) >= chunkSize {
			fmt.Printf("Adding chunk.\n")
			update, err := index.AddDocuments(currentChunk, "id")
			if err != nil {
				panic(err)
			}
			updates = append(updates, update.TaskUID)
			currentChunk = []map[string]interface{}{}
		}
	}
	if len(currentChunk) > 0 {
		update, err := index.AddDocuments(currentChunk, "id")
		if err != nil {
			panic(err)
		}
		updates = append(updates, update.TaskUID)
	}
	fmt.Printf("Waiting for updates...\n")

	for _, update := range updates {
		index.WaitForTask(update)
	}

	fmt.Printf("Completed.\n")

	return nil
}

func main() {
	ctx := context.Background()
	courses, _, err := vvzfetch.FetchAllCourses(ctx, "2022W", "en")
	if err != nil {
		log.Fatal(err)
	}
	insertIntoIndex("courses", courses)
}

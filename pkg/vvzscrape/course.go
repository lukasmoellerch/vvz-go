package vvzscrape

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// regex to extract ids from /Vorlesungsverzeichnis/lerneinheit.view?lerneinheitId=148661&semkez=2021W&ansicht=LEHRVERANSTALTUNGEN&lang=de
var courseRegex = regexp.MustCompile(`\/Vorlesungsverzeichnis\/lerneinheit\.view\?lerneinheitId=(\d+)&semkez=[A-Z0-9]+&ansicht=LEHRVERANSTALTUNGEN&lang=en`)

type Course struct {
	Id         int         `json:"id,omitempty"`
	Segments   [][]string  `json:"segments,omitempty"`
	ReadableId string      `json:"readable_id,omitempty"`
	Title      string      `json:"title,omitempty"`
	CourseType string      `json:"course_type,omitempty"`
	Ects       string      `json:"ects,omitempty"`
	Hours      string      `json:"hours,omitempty"`
	Lecturer   []*Lecturer `json:"lecturer,omitempty"`
	Abstract   string      `json:"abstract,omitempty"`
	Objective  string      `json:"objective,omitempty"`
	Content    string      `json:"content,omitempty"`
	Notice     string      `json:"notice,omitempty"`
}

// items, _ := ioutil.ReadDir("./data/courses")
/*
	if item.IsDir() {
		continue
	}
	file, err := os.Open("./data/courses/" + item.Name())
	if err != nil {
		panic(err)
	}
*/
func ScrapeCourses(lecturers map[int]*Lecturer, reader io.Reader) (map[string]*Course, error) {
	courses := make(map[string]*Course)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		panic(err)
	}
	mainTable := doc.Find("table").First()
	body := mainTable.ChildrenFiltered("tbody").First()
	rows := body.ChildrenFiltered("tr")

	currentCourse := ""
	levels := make([]string, 0)

	var rowError error
	rows.EachWithBreak(func(i int, s *goquery.Selection) bool {
		cells := s.ChildrenFiltered("td")
		firstCell := cells.First()
		if firstCell.AttrOr("class", "") == "td-level" {
			indicators := firstCell.ChildrenFiltered("img.levelIndicator").Length()

			levels = levels[:indicators]

			link := firstCell.ChildrenFiltered("a").First()
			levels = append(levels, link.Text())

		} else if firstCell.AttrOr("style", "") == "border-top:1px solid #ccc;" {
			courseLecturers := make([]*Lecturer, 0)
			var courseId int
			var readableId, title, courseType, ects, hours string

			cells.EachWithBreak(func(i int, s *goquery.Selection) bool {
				switch i {
				case 0:
					readableId = s.Text()
				case 1:
					titleContainer := s.ChildrenFiltered("b").First()
					link := titleContainer.ChildrenFiltered("a").First()
					// href has format /Vorlesungsverzeichnis/lerneinheit.view?lerneinheitId=148661&semkez=2021W&ansicht=LEHRVERANSTALTUNGEN&lang=de
					// We want only the id (148661)
					href := link.AttrOr("href", "")
					// Use a regex to extract the id
					matches := courseRegex.FindStringSubmatch(href)
					courseId = -1
					if len(matches) > 0 {
						courseIdString := matches[1]
						// Convert to int
						courseId, err = strconv.Atoi(courseIdString)
						if err != nil {
							courseId = -1
						}
					}
					title = titleContainer.Text()
				case 2:
					courseType = s.Text()
				case 3:
					ects = s.Text()
				case 4:
					hours = s.Text()
				case 5:
					aElems := s.ChildrenFiltered("a")
					aElems.EachWithBreak(func(i int, aElem *goquery.Selection) bool {

						href := aElem.AttrOr("href", "")
						// Extract id from href:
						matches := dozRegex.FindStringSubmatch(href)
						lecturerId := -1
						if len(matches) > 0 {
							lecturerId, err = strconv.Atoi(matches[1])
							if err != nil {
								lecturerId = -1
							}
						} else {
							rowError = fmt.Errorf("could not extract lecturer id from href: %s", href)
							return false
						}
						// Get lecturer from map
						l, ok := lecturers[lecturerId]
						if !ok {
							rowError = fmt.Errorf("could not find lecturer with id %d", lecturerId)
							return false
						}
						courseLecturers = append(courseLecturers, l)
						return true
					})
					if rowError != nil {
						return false
					}

				}
				return true
			})
			if rowError != nil {
				return false
			}
			existingCourse, ok := courses[readableId]
			levelsCopy := make([]string, len(levels))
			copy(levelsCopy, levels)
			if ok {
				existingCourse.Segments = append(existingCourse.Segments, levelsCopy)
				currentCourse = ""
			} else {
				currentCourse = readableId
				courses[currentCourse] = &Course{
					Id:         courseId,
					Segments:   [][]string{levelsCopy},
					ReadableId: readableId,
					Title:      title,
					CourseType: courseType,
					Ects:       ects,
					Hours:      hours,
					Lecturer:   courseLecturers,
				}
			}
		} else if currentCourse != "" && firstCell.Text() == "Abstract" {
			secondCell := firstCell.Next()
			courses[currentCourse].Abstract = secondCell.Text()
		} else if currentCourse != "" && firstCell.Text() == "Objective" {
			secondCell := firstCell.Next()
			courses[currentCourse].Objective = secondCell.Text()
		} else if currentCourse != "" && firstCell.Text() == "Content" {
			secondCell := firstCell.Next()
			courses[currentCourse].Content = secondCell.Text()
		} else if currentCourse != "" && firstCell.Text() == "Notice" {
			secondCell := firstCell.Next()
			courses[currentCourse].Notice = secondCell.Text()
		}
		return true
	})
	if rowError != nil {
		return nil, rowError
	}

	return courses, nil
}

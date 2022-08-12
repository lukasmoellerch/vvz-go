package vvzscrape

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// http://www.vvz.ethz.ch/Vorlesungsverzeichnis/dozent.view?dozide=10055212&semkez=2021W&lang=en
var dozRegex = regexp.MustCompile(`\/Vorlesungsverzeichnis\/dozent\.view\?dozide=(\d+)&.+`)

type Lecturer struct {
	Id int `json:"id,omitempty"`

	LastName   string `json:"last_name,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	Title      string `json:"title,omitempty"`
	Field      string `json:"field,omitempty"`
	Department string `json:"department,omitempty"`
}

func ScrapeLecturers(reader io.Reader) (map[int]*Lecturer, error) {
	lecturers := make(map[int]*Lecturer)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		panic(err)
	}

	mainTable := doc.Find("table").First()
	body := mainTable.ChildrenFiltered("tbody").First()
	rows := body.ChildrenFiltered("tr")

	var rowError error
	rows.EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		cells := s.ChildrenFiltered("td")

		lastnameCell := cells.First()
		bElem := lastnameCell.ChildrenFiltered("b").First()
		aElem := bElem.ChildrenFiltered("a").First()
		href := aElem.AttrOr("href", "")
		// Extract id from href:
		matches := dozRegex.FindStringSubmatch(href)
		id := -1
		if len(matches) > 0 {
			id, err = strconv.Atoi(matches[1])
			if err != nil {
				id = -1
			}
		} else {
			s, _ := s.Html()
			rowError = fmt.Errorf("could not extract id from href: %s", s)
			return false
		}

		firstnameCell := lastnameCell.Next()
		titleCell := firstnameCell.Next()
		fieldCell := titleCell.Next()
		departmentCell := fieldCell.Next()
		lecturer := &Lecturer{
			Id:         id,
			LastName:   lastnameCell.Text(),
			FirstName:  firstnameCell.Text(),
			Title:      titleCell.Text(),
			Field:      fieldCell.Text(),
			Department: departmentCell.Text(),
		}

		lecturers[id] = lecturer
		return true
	})
	if rowError != nil {
		return nil, rowError
	}

	return lecturers, nil
}

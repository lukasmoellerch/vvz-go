package vvzfetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/lukasmoellerch/vvz-go/pkg/vvzscrape"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	hc http.Client
}

type FetchLecturersOptions struct {
	Semkez string
	Lang   string
}

const lecturerUrl = "http://www.vvz.ethz.ch/Vorlesungsverzeichnis/sucheDozierende.view"

func (c *Client) FetchLecturers(ctx context.Context, options FetchLecturersOptions) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", lecturerUrl, nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	query.Add("semkez", "2022W")
	query.Add("seite", "0")
	query.Add("lang", "en")
	req.URL.RawQuery = query.Encode()

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type FetchCoursesOptions struct {
	Semkez     string
	Lang       string
	View       string
	DegreeType string
}

const courseUrl = "http://www.vvz.ethz.ch/Vorlesungsverzeichnis/sucheLehrangebot.view"

func (c *Client) FetchCourses(ctx context.Context, options FetchCoursesOptions) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", courseUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	query := req.URL.Query()
	query.Add("lang", options.Lang)
	query.Add("semkez", options.Semkez)
	query.Add("ansicht", options.View)
	query.Add("studiengangTyp", options.DegreeType)
	query.Add("seite", "0")
	req.URL.RawQuery = query.Encode()

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	return resp, nil

}

func FetchAllCourses(ctx context.Context, semkez, lang string) (map[string]*vvzscrape.Course, map[int]*vvzscrape.Lecturer, error) {
	c := Client{}
	var lecturers map[int]*vvzscrape.Lecturer
	eg, egctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		resp, err := c.FetchLecturers(egctx, FetchLecturersOptions{
			Semkez: semkez,
			Lang:   lang,
		})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		lecturers, err = vvzscrape.ScrapeLecturers(resp.Body)
		return err
	})

	degreeTypes := []string{"BSC", "DZ", "DS", "DR", "SHE", "MSC", "GS", "WBZ", "NDS"}

	bodies := make([][]byte, len(degreeTypes))
	for i, t := range degreeTypes {
		i := i
		t := t
		eg.Go(func() error {
			resp, err := c.FetchCourses(ctx, FetchCoursesOptions{
				Semkez:     semkez,
				Lang:       lang,
				View:       "2",
				DegreeType: t,
			})
			if err != nil {
				return err
			}
			bodies[i], err = io.ReadAll(resp.Body)
			return err
		})

	}
	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	courses := make(map[string]*vvzscrape.Course)
	for _, body := range bodies {

		courseMap, err := vvzscrape.ScrapeCourses(lecturers, bytes.NewReader(body))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scrape courses: %s", err.Error())
		}
		for _, course := range courseMap {
			courses[course.ReadableId] = course
		}
	}

	return courses, lecturers, nil
}

package ics

import (
	"bufio"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/brimstone/logger"
)

type Event struct {
	Attendee string
	Duration time.Duration
	End      time.Time
	Raw      []string
	Start    time.Time
	Summary  string
}

func parseTime(attributes []string, t string) (time.Time, error) {
	log := logger.New()
	var actual time.Time
	var err error
	if len(attributes) == 1 {
		actual, err = time.Parse("20060102T150405Z", t)
		if err == nil {
			if actual.IsZero() {
				log.Error("Failed to parse",
					log.Field("time", t),
				)
			}
			return actual, nil
		}
		actual, err = time.Parse("20060102T150405", t)
		if err == nil {
			if actual.IsZero() {
				log.Error("Failed to parse",
					log.Field("time", t),
				)
			}
			return actual, nil
		}
		return actual, err
	}
	if attributes[1] == "VALUE=DATE-TIME" {
		actual, err = time.Parse("20060102T150405Z", t)
		if err != nil {
			return actual, err
		}
	} else if attributes[1][0:5] == "TZID=" {
		l, err := time.LoadLocation(attributes[1][5:])
		if err != nil {
			return actual, err
		}
		actual, err = time.ParseInLocation("20060102T150405", t, l)
		if err != nil {
			return actual, err
		}
	} else if attributes[1] == "VALUE=DATE" {
		actual, err = time.Parse("20060102", t)
		if err != nil {
			return actual, err
		}
	}
	return actual, err
}

func ReaderEvents(r io.Reader) ([]Event, error) {
	//log := logger.New()
	var events []Event
	reader := bufio.NewReader(r)
	var e Event
	var text string
	var raw []string
	for {
		t, err := reader.ReadString('\n')
		raw = append(raw, t)
		if err != nil {
			break
		}
		text += strings.Trim(t, " \r\n")
		peek, err := reader.Peek(1)
		if err == nil && peek[0] == ' ' {
			continue
		}

		fields := strings.SplitN(text, ":", 2)
		// Clear text for next pass
		text = ""
		attributes := strings.Split(fields[0], ";")
		switch attributes[0] {
		case "BEGIN":
			if fields[1] == "VEVENT" {
				e = Event{}
				// trim raw log to just the last line
				raw = []string{raw[len(raw)-1]}
			}

		case "END":
			if fields[1] == "VEVENT" {
				e.Duration = e.End.Sub(e.Start)
				e.Raw = raw
				events = append(events, e)
			}

		case "SUMMARY":
			e.Summary = fields[1]

		case "ATTENDEE":
			e.Attendee = strings.TrimPrefix(fields[1], "mailto:")

		case "DTSTART":
			e.Start, err = parseTime(attributes, fields[1])
			if err != nil {
				return events, err
			}

		case "DTEND":
			e.End, err = parseTime(attributes, fields[1])
			if err != nil {
				return events, err
			}
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})
	return events, nil

}

func HTTPEvents(url string) ([]Event, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	f := resp.Body
	return ReaderEvents(f)
}

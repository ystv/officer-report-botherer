package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var motivationalMessages = []string{
	"COMMIT YOUR GLORIOUS TALE TO THE ANNALS OF YSTV HISTORY!",
	"Beep boop. Please write a bit about what you've been up to this week.",
	"I'm sorry dave, I can't let you not write an officer report",
	"Tell us what you've been up to this week!",
	"Extra points for memes",
	"Oh hey sorry to bother you, just a quick thought: you really should write up an officer report",
}

const offRepStatusQuery = `
	SELECT first_name, last_name, contents IS NOT NULL AS written FROM
		member_officerships LEFT JOIN members ON
			member_id=members.id
		LEFT JOIN officer_reports ON (
			member_officerships.id=member_officership_id AND
			officer_reports.created_date > $1
		) WHERE
			start_date < NOW() AND (
				end_date IS NULL OR end_date > NOW()
			)`

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err.Error())
	}
}

func calculateLastMeeting(db *sqlx.DB, relativeTo time.Time, meetingDay time.Weekday, meetingHour int) (lastMeeting time.Time) {
	var termstart time.Time
	check(
		db.Get(&termstart, "select start_date from term_dates where start_date < NOW() AND (start_date + interval '10 weeks') > NOW() LIMIT 1"),
		"Error getting term start date",
	)

	//weeknum := int(time.Now().Sub(termstart) / (time.Hour * 24 * 7))
	lastMeeting = relativeTo
	if !(lastMeeting.Weekday() == meetingDay && lastMeeting.Hour() >= meetingHour) {
		// If we are not on a meeting day or on a meeting day before the time of the meeting
		for lastMeeting.Weekday() != meetingDay {
			// REWIND TIME!
			lastMeeting = lastMeeting.Add(-time.Hour * 24)
		}
	}
	lastMeeting = time.Date(lastMeeting.Year(), lastMeeting.Month(), lastMeeting.Day(), meetingHour, 0, 0, 0, lastMeeting.Location())

	return
}
func main() {
	rand.Seed(time.Now().UnixNano())

	dbURL := flag.String("dburl", "", "Database URL")
	webhookurl := flag.String("webhookurl", "", "Slack Webhook URL")
	flag.Parse()

	if !strings.HasPrefix(*dbURL, "postgres://") {
		*dbURL = "postgres://" + *dbURL
	}
	db := sqlx.MustConnect("postgres", *dbURL)

	lastMeeting := calculateLastMeeting(db, time.Now(), time.Tuesday, 19)

	var offRepStatus []struct {
		FirstName string `db:"first_name"`
		LastName  string `db:"last_name"`
		Written   bool   `db:"written"`
	}

	err := db.Select(&offRepStatus, offRepStatusQuery, lastMeeting)
	if err != nil {
		log.Fatal("Error retrieving officers: ", err)
	}

	var written, unWritten string

	for _, o := range offRepStatus {
		if o.Written {
			written += fmt.Sprintf("✅ %s %s\n", o.FirstName, o.LastName)
		} else {
			unWritten += fmt.Sprintf("❓ %s %s\n", o.FirstName, o.LastName)
		}
	}

	type attachment struct {
		Color string `json:"color"`
		Title string `json:"title"`
		Text  string `json:"text"`
	}

	jsonData, err := json.Marshal(struct {
		Text        string       `json:"text"`
		Attachments []attachment `json:"attachments"`
	}{
		motivationalMessages[rand.Intn(len(motivationalMessages))],
		[]attachment{
			{"good", "Report written", written},
			{"warning", "Report not yet written", unWritten},
		},
	})
	if err != nil {
		log.Fatal("Error marshalling json: ", err)
	}

	res, err := http.Post(*webhookurl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal("Error posting to slack: ", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		log.Fatal("Error posting to slack: non-200 response: ", res.Status)
	}
}

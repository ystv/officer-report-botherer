# Officer Report Nagging Bot

An irritating little utility that looks up who hasn't written an officer report since the last station meeting, and posts a list to slack.

## Usage
`officer-report-botherer -dburl user:pass@dbhost/dbname -webhookurl "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"`

Run periodically using cron on a server :)

## TODO
- Currently hardcoded meeting time of Tuesdays, 19:00
- Happily runs out of term time...
- Probably does weird $#!t with timezones
- Tests for the above...

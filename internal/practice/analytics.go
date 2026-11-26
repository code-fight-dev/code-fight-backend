package practice

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dateLayout = "2006-01-02"

type TaskCounts struct {
	Key            string `json:"key"`
	AttemptedTasks int    `json:"attemptedTasks"`
	SolvedTasks    int    `json:"solvedTasks"`
	AvailableTasks int    `json:"availableTasks"`
}

type LanguageCounts struct {
	Language            string `json:"language"`
	Submissions         int    `json:"submissions"`
	AcceptedSubmissions int    `json:"acceptedSubmissions"`
	AttemptedTasks      int    `json:"attemptedTasks"`
	SolvedTasks         int    `json:"solvedTasks"`
}

type DailyCounts struct {
	Date        string `json:"date"`
	Submissions int    `json:"submissions"`
	FirstSolves int    `json:"firstSolves"`
}

type Summary struct {
	Totals struct {
		Submissions    int `json:"submissions"`
		AttemptedTasks int `json:"attemptedTasks"`
		SolvedTasks    int `json:"solvedTasks"`
	} `json:"totals"`
	Activity struct {
		From     string        `json:"from"`
		To       string        `json:"to"`
		TimeZone string        `json:"timeZone"`
		Days     []DailyCounts `json:"days"`
	} `json:"activity"`
	Streaks struct {
		CurrentDays int `json:"currentDays"`
		LongestDays int `json:"longestDays"`
	} `json:"streaks"`
	ByDifficulty []TaskCounts     `json:"byDifficulty"`
	ByCategory   []TaskCounts     `json:"byCategory"`
	ByTag        []TaskCounts     `json:"byTag"`
	ByLanguage   []LanguageCounts `json:"byLanguage"`
}

type Submission struct {
	SubmissionID    uuid.UUID `json:"submissionId"`
	TaskSlug        string    `json:"taskSlug"`
	TaskTitle       string    `json:"taskTitle"`
	Difficulty      string    `json:"difficulty"`
	Language        string    `json:"language"`
	LanguageVersion string    `json:"languageVersion"`
	Status          string    `json:"status"`
	Verdict         *string   `json:"verdict"`
	SubmittedAt     time.Time `json:"submittedAt"`
}

type historyCursor struct {
	Time time.Time `json:"time"`
	ID   uuid.UUID `json:"id"`
}

func RegisterRoutes(router fiber.Router, db *pgxpool.Pool) {
	router.Get("/summary", func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(uuid.UUID)
		if userID == uuid.Nil {
			return fiber.ErrUnauthorized
		}
		zone := c.Query("timeZone", "UTC")
		location, err := time.LoadLocation(zone)
		if err != nil || zone == "Local" || zone == "" {
			return fiber.NewError(fiber.StatusBadRequest, "invalid timeZone")
		}
		today, _ := time.Parse(dateLayout, time.Now().In(location).Format(dateLayout))
		to, err := time.Parse(dateLayout, c.Query("to", today.Format(dateLayout)))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid to date")
		}
		from, err := time.Parse(dateLayout, c.Query("from", to.AddDate(0, 0, -364).Format(dateLayout)))
		if err != nil || from.Before(to.AddDate(0, 0, -365)) || from.After(to) || to.After(today) || from.Year() < 1 {
			return fiber.NewError(fiber.StatusBadRequest, "date range must contain 1 to 366 days and end by today")
		}
		var summary Summary
		if err := db.QueryRow(c.UserContext(), summarySQL, userID, zone, from.Format(dateLayout), to.Format(dateLayout), today.Format(dateLayout)).Scan(&summary); err != nil {
			return err
		}
		return c.JSON(summary)
	})
	router.Get("/submissions", func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(uuid.UUID)
		if userID == uuid.Nil {
			return fiber.ErrUnauthorized
		}
		limit, err := strconv.Atoi(c.Query("limit", "20"))
		if err != nil || limit < 1 || limit > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "limit must be between 1 and 100")
		}
		var after historyCursor
		var before *time.Time
		if raw := c.Query("cursor"); raw != "" {
			data, err := base64.RawURLEncoding.DecodeString(raw)
			if len(raw) > 256 || err != nil || json.Unmarshal(data, &after) != nil || after.Time.IsZero() || after.ID == uuid.Nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid cursor")
			}
			before = &after.Time
		}
		language := strings.ToLower(strings.TrimSpace(c.Query("language")))
		verdict := strings.ToLower(strings.TrimSpace(c.Query("verdict")))
		rows, err := db.Query(c.UserContext(), historySQL, userID, language, verdict, before, after.ID, limit+1)
		if err != nil {
			return err
		}
		items, err := pgx.CollectRows(rows, pgx.RowToStructByName[Submission])
		if err != nil {
			return err
		}
		page := struct {
			Items      []Submission `json:"items"`
			NextCursor string       `json:"nextCursor,omitempty"`
		}{Items: items}
		if len(items) > limit {
			last := items[limit-1]
			data, err := json.Marshal(historyCursor{Time: last.SubmittedAt.UTC(), ID: last.SubmissionID})
			if err != nil {
				return err
			}
			page.Items = items[:limit]
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
		}
		return c.JSON(page)
	})
}

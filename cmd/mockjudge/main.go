package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SubmitRequest struct {
	SubmissionID    string `json:"submission_id"`
	TaskID          string `json:"task_id"`
	Language        string `json:"language"`
	LanguageVersion string `json:"language_version"`
	SourceCode      string `json:"source_code"`
	CallbackURL     string `json:"callback_url"`
	CallbackToken   string `json:"callback_token"`
}

type SubmitResponse struct {
	JudgeSubmissionID string `json:"judge_submission_id"`
	Status            string `json:"status"`
}

type CallbackPayload struct {
	JudgeSubmissionID string  `json:"judge_submission_id"`
	SubmissionID      string  `json:"submission_id"`
	JudgeStatus       string  `json:"judge_status"`
	Verdict           *string `json:"verdict"`
	PassedTests       int     `json:"passed_tests"`
	TotalTests        int     `json:"total_tests"`
	Score             int     `json:"score"`
}

func main() {
	app := fiber.New()

	app.Post("/submissions", func(c *fiber.Ctx) error {
		var req SubmitRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).SendString(err.Error())
		}

		log.Println("judge got submission:", req.SubmissionID)

		judgeID := uuid.NewString()

		go func() {
			time.Sleep(2 * time.Second)

			verdict := "accepted"
			payload := CallbackPayload{
				JudgeSubmissionID: judgeID,
				SubmissionID:      req.SubmissionID,
				JudgeStatus:       "finished",
				Verdict:           &verdict,
				PassedTests:       3,
				TotalTests:        3,
				Score:             100,
			}

			body, _ := json.Marshal(payload)

			httpReq, _ := http.NewRequest(http.MethodPost, req.CallbackURL, bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("X-Judge-Token", req.CallbackToken)

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				log.Println("judge callback error:", err)
				return
			}
			defer resp.Body.Close()

			log.Println("judge callback status:", resp.StatusCode)
		}()

		return c.JSON(SubmitResponse{
			JudgeSubmissionID: judgeID,
			Status:            "queued",
		})
	})

	log.Fatal(app.Listen(":8090"))
}
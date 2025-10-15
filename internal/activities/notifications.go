package activities

import (
"bytes"
"context"
"encoding/json"
"fmt"
"log"
"net/http"
"os"
"time"
)

type NotificationPayload struct {
Input struct {
UserID    string            `json:"user_id"`
RequestID string            `json:"request_id"`
Model     string            `json:"model"`
Prompt    string            `json:"prompt"`
Params    map[string]string `json:"params"`
} `json:"input"`
Result struct {
Output string            `json:"output"`
Meta   map[string]string `json:"meta"`
} `json:"result"`
}

// NotifyUI пушить подію в Webhook або WS Hub. Для прикладу — HTTP POST у webhook.
func NotifyUI(ctx context.Context, payload NotificationPayload) error {
webhookURL := os.Getenv("NOTIFICATIONS_WEBHOOK_URL")
if webhookURL == "" {
log.Printf("no NOTIFICATIONS_WEBHOOK_URL configured, skipping notification for %s:%s", payload.Input.UserID, payload.Input.RequestID)
return nil
}

body, err := json.Marshal(struct {
UserID    string            `json:"user_id"`
RequestID string            `json:"request_id"`
Status    string            `json:"status"`
Result    map[string]string `json:"result_meta"`
}{
UserID:    payload.Input.UserID,
RequestID: payload.Input.RequestID,
Status:    "completed",
Result:    payload.Result.Meta,
})
if err != nil {
return err
}

req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
if err != nil {
return err
}
req.Header.Set("Content-Type", "application/json")

client := &http.Client{Timeout: 5 * time.Second}
resp, err := client.Do(req)
if err != nil {
return err
}
defer resp.Body.Close()

if resp.StatusCode >= 300 {
return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

return nil
}


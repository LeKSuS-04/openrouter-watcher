# OpenRouter Watcher

Dead simple tool to monitor your OpenRouter credits.

Configuration is done via environment variables:

| Environment Variable      | Description                  | Default                                | Required |
| ------------------------- | ---------------------------- | -------------------------------------- | -------- |
| `OPENROUTER_API_TOKEN`    | Your OpenRouter API token    | -                                      | Yes      |
| `OPENROUTER_API_ENDPOINT` | Your OpenRouter API endpoint | `https://openrouter.ai/api/v1/credits` | No       |
| `WATCHER_INTERVAL`        | Interval to check credits    | `15s`                                  | No       |
| `EXPORTER_ADDRESS`        | Address to expose metrics    | `:9080`                                | No       |
| `EXPORTER_ENDPOINT`       | Endpoint to expose metrics   | `/metrics`                             | No       |

Following sensors are exposed:

- `openrouter_total_credits{}`
- `openrouter_total_usage{}`
- `openrouter_api_request_duration{}`
- `openrouter_api_successful_requests{}`
- `openrouter_api_failed_requests{error_code}`

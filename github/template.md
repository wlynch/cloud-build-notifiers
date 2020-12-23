| Build Information |   |
| ----------------- | - |
| Status  | **{{.GetStatus}} {{ emoji .GetStatus }}** |
| Trigger | [{{.BuildTriggerId}}](https://console.cloud.google.com/cloud-build/triggers/{{.BuildTriggerId}})
| Image   | {{ range .Results.GetImages }}{{.Name}} {{end}} |
| Start   | {{ unix .StartTime }} |
| Duration | {{ duration .StartTime .FinishTime }} |

#### Steps
| Step | Status | Duration |
| ---- | ------ | -------- |
{{ range .Steps }}| {{.Name}} | {{.Status}} | {{ duration .Timing.StartTime .Timing.EndTime }} |
{{end}}
package main

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	pb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
)

var (
	summaryTmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"emoji":    emoji,
		"unix":     unixFormat,
		"duration": duration,
	}).ParseFiles("template.md"))
)

func renderTemplate(build *pb.Build) (string, error) {
	b := new(bytes.Buffer)
	if err := summaryTmpl.ExecuteTemplate(b, "template.md", build); err != nil {
		return "", fmt.Errorf("template.Execute: %w", err)
	}
	return b.String(), nil
}

func emoji(s pb.Build_Status) string {
	switch s {
	case pb.Build_QUEUED:
		return ":clock1:"
	case pb.Build_WORKING:
		return ":runner:"
	case pb.Build_SUCCESS:
		return ":white_check_mark:"
	case pb.Build_FAILURE, pb.Build_INTERNAL_ERROR, pb.Build_STATUS_UNKNOWN:
		return ":x:"
	case pb.Build_CANCELLED:
		return ":no_entry_sign:"
	case pb.Build_TIMEOUT, pb.Build_EXPIRED:
		return ":skull:"
	}
	return ":shipit:"
}

func unixFormat(ts *tspb.Timestamp) string {
	return unixTS(ts).Format(time.RFC3339)
}

func unixTS(ts *tspb.Timestamp) time.Time {
	return time.Unix(ts.GetSeconds(), int64(ts.GetNanos()))
}

func duration(t1, t2 *tspb.Timestamp) time.Duration {
	return unixTS(t2).Sub(unixTS(t1)).Round(time.Millisecond)
}

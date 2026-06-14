// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"oblikovati.org/build"
	"oblikovati.org/update"
)

// updateTimeout bounds the GitHub query so `check-updates` returns promptly on a slow or
// missing network instead of hanging the headless tool.
const updateTimeout = 8 * time.Second

// cmdCheckUpdates queries GitHub for a newer release on this build's channel and prints a
// plain-text result (CLAUDE.md: plain text for user-facing CLI output). An offline
// machine or a dev build is reported as a graceful skip, not an error.
//
//	oblikovati-cli check-updates
func cmdCheckUpdates(out io.Writer) error {
	src := update.NewGitHubSource(update.DefaultOwner, update.DefaultRepo, &http.Client{Timeout: updateTimeout})
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	res, err := update.NewChecker(src).Check(ctx, build.Version)
	if err != nil {
		return fmt.Errorf("oblikovati-cli: update check failed: %w", err)
	}
	fmt.Fprintln(out, updateMessage(res))
	return nil
}

// updateMessage renders a check result as one line of user-facing text. Kept pure (no
// I/O) so it is table-testable without a live GitHub.
func updateMessage(res update.Result) string {
	if res.Skipped {
		return "Update check skipped: " + skipDetail(res.SkipReason, res.Channel) + "."
	}
	if res.UpdateAvailable {
		return fmt.Sprintf("A new %s release is available: %s\nDownload: %s",
			res.Channel, res.Latest.Version, res.Latest.HTMLURL)
	}
	return fmt.Sprintf("You are running the latest %s release (%s).", res.Channel, res.Current)
}

// skipDetail expands a skip reason into a full clause for the CLI message.
func skipDetail(reason string, channel update.Channel) string {
	switch reason {
	case "offline":
		return "no internet connection"
	case "development build":
		return "this is a development build"
	case "no published release":
		return "no release published on the " + channel.String() + " channel yet"
	default:
		return reason
	}
}

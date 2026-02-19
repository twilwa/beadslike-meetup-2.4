// ABOUTME: Tests for tl claim command — verifies open->in_progress claim behavior and output.
// ABOUTME: Covers not-found/not-open errors and concurrent claim contention with exactly one winner.

package tl

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimCmdArgsRequireSingleID(t *testing.T) {
	require.NotNil(t, claimCmd.Args)
	assert.Error(t, claimCmd.Args(claimCmd, []string{}))
	assert.NoError(t, claimCmd.Args(claimCmd, []string{"tl-001"}))
	assert.Error(t, claimCmd.Args(claimCmd, []string{"tl-001", "extra"}))
}

func TestClaimOpenTask(t *testing.T) {
	dir := seedIssue(t, "tl-claim-01", "Claimable", StatusOpen)
	setClaimCommandGlobals(t, dir, false, "agent-1")

	cmd := newClaimCommand(t)
	err := runClaim(cmd, []string{"tl-claim-01"})
	require.NoError(t, err)
	assert.Equal(t, "Claimed tl-claim-01 by agent-1\n", cmd.OutOrStdout().(*bytes.Buffer).String())

	g, err := loadGraph(dir)
	require.NoError(t, err)
	issue := g.Tasks["tl-claim-01"]
	require.NotNil(t, issue)
	assert.Equal(t, StatusInProgress, issue.Status)
	assert.Equal(t, "agent-1", issue.Assignee)
}

func TestClaimJSONOutput(t *testing.T) {
	dir := seedIssue(t, "tl-claim-02", "JSON Claim", StatusOpen)
	setClaimCommandGlobals(t, dir, true, "agent-json")

	cmd := newClaimCommand(t)
	err := runClaim(cmd, []string{"tl-claim-02"})
	require.NoError(t, err)

	var issue Issue
	require.NoError(t, json.Unmarshal(cmd.OutOrStdout().(*bytes.Buffer).Bytes(), &issue))
	assert.Equal(t, "tl-claim-02", issue.ID)
	assert.Equal(t, StatusInProgress, issue.Status)
	assert.Equal(t, "agent-json", issue.Assignee)
}

func TestClaimNotFound(t *testing.T) {
	dir := seedIssue(t, "tl-claim-03", "Exists", StatusOpen)
	setClaimCommandGlobals(t, dir, false, "agent-1")

	err := runClaim(newClaimCommand(t), []string{"tl-missing"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestClaimNotOpen(t *testing.T) {
	dir := seedIssue(t, "tl-claim-04", "Already Claimed", StatusInProgress)
	setClaimCommandGlobals(t, dir, false, "agent-2")

	err := runClaim(newClaimCommand(t), []string{"tl-claim-04"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not open")
}

func TestClaimConcurrentOnlyOneSucceeds(t *testing.T) {
	dir := seedIssue(t, "tl-claim-05", "Contended", StatusOpen)
	setClaimCommandGlobals(t, dir, false, "agent-race")

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- runClaim(newClaimCommand(t), []string{"tl-claim-05"})
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	var successes int
	var failures []error
	for err := range errCh {
		if err == nil {
			successes++
			continue
		}
		failures = append(failures, err)
	}

	require.Equal(t, 1, successes)
	require.Len(t, failures, 1)
	assert.True(
		t,
		errors.Is(failures[0], ErrLockBusy) || strings.Contains(failures[0].Error(), "not open"),
		"expected lock busy or not open error, got: %v", failures[0],
	)

	g, err := loadGraph(dir)
	require.NoError(t, err)
	issue := g.Tasks["tl-claim-05"]
	require.NotNil(t, issue)
	assert.Equal(t, StatusInProgress, issue.Status)
	assert.Equal(t, "agent-race", issue.Assignee)
}

func newClaimCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd
}

func setClaimCommandGlobals(t *testing.T, dir string, json bool, agent string) {
	t.Helper()
	prevJSON := jsonOutput
	prevDir := tlDirFlag
	prevAgent := claimAgent
	t.Cleanup(func() {
		jsonOutput = prevJSON
		tlDirFlag = prevDir
		claimAgent = prevAgent
	})

	jsonOutput = json
	tlDirFlag = dir
	claimAgent = agent
}

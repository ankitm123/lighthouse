/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dco

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/prow/fakegithub"
	"github.com/jenkins-x/lighthouse/pkg/prow/github"
)

type fakePruner struct{}

func (fp *fakePruner) PruneComments(shouldPrune func(scm.Comment) bool) {}

func strP(str string) *string {
	return &str
}

func TestHandlePullRequest(t *testing.T) {
	var testcases = []struct {
		// test settings
		name string

		// PR settings
		pullRequestEvent scm.PullRequestHook
		commits          []scm.RepositorysitoryCommit
		issueState       string
		hasDCOYes        bool
		hasDCONo         bool
		// status of the DCO github context
		status string

		// expectations
		addedLabel     string
		removedLabel   string
		expectedStatus string
		// org/repo#number:body
		addedComment string
		// org/repo#issuecommentid
		removedComment string
	}{
		{
			name: "should not do anything on pull request edited",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionEdited,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
		},
		{
			name: "should add 'no' label & status context and add a comment if no commits have sign off",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionOpen,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha", Commit: github.GitCommit{Message: "not a sign off"}},
			},
			issueState: "open",
			hasDCONo:   false,
			hasDCOYes:  false,

			addedLabel:     fmt.Sprintf("/#3:%s", dcoNoLabel),
			expectedStatus: github.StatusFailure,
			addedComment: `/#3:Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits.

:memo: **Please follow instructions in the [contributing guide](https://github.com///blob/master/CONTRIBUTING.md) to update your commits with the DCO**

Full details of the Developer Certificate of Origin can be found at [developercertificate.org](https://developercertificate.org/).

**The list of commits missing DCO signoff**:

- [sha](https://github.com///commits/sha) not a sign off

<details>

Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).
</details>
`,
		},
		{
			name: "should add 'no' label & status context, remove old labels and add a comment if no commits have sign off",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionOpen,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha", Commit: github.GitCommit{Message: "not a sign off"}},
			},
			issueState: "open",
			hasDCONo:   false,
			hasDCOYes:  true,

			addedLabel:     fmt.Sprintf("/#3:%s", dcoNoLabel),
			removedLabel:   fmt.Sprintf("/#3:%s", dcoYesLabel),
			expectedStatus: github.StatusFailure,
			addedComment: `/#3:Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits.

:memo: **Please follow instructions in the [contributing guide](https://github.com///blob/master/CONTRIBUTING.md) to update your commits with the DCO**

Full details of the Developer Certificate of Origin can be found at [developercertificate.org](https://developercertificate.org/).

**The list of commits missing DCO signoff**:

- [sha](https://github.com///commits/sha) not a sign off

<details>

Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).
</details>
`,
		},
		{
			name: "should update comment if labels and status are up to date and sign off is failing",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionOpen,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha", Commit: github.GitCommit{Message: "not a sign off"}},
			},
			issueState: "open",
			hasDCONo:   true,
			hasDCOYes:  false,
			status:     github.StatusFailure,

			expectedStatus: github.StatusFailure,
			addedComment: `/#3:Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits.

:memo: **Please follow instructions in the [contributing guide](https://github.com///blob/master/CONTRIBUTING.md) to update your commits with the DCO**

Full details of the Developer Certificate of Origin can be found at [developercertificate.org](https://developercertificate.org/).

**The list of commits missing DCO signoff**:

- [sha](https://github.com///commits/sha) not a sign off

<details>

Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).
</details>
`,
		},
		{
			name: "should mark the PR as failed if just one commit is missing sign-off",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionOpen,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha1", Commit: github.GitCommit{Message: "Signed-off-by: someone"}},
				{SHA: "sha", Commit: github.GitCommit{Message: "not signed off"}},
			},
			issueState: "open",
			hasDCONo:   false,
			hasDCOYes:  true,

			addedLabel:     fmt.Sprintf("/#3:%s", dcoNoLabel),
			removedLabel:   fmt.Sprintf("/#3:%s", dcoYesLabel),
			expectedStatus: github.StatusFailure,
			addedComment: `/#3:Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits.

:memo: **Please follow instructions in the [contributing guide](https://github.com///blob/master/CONTRIBUTING.md) to update your commits with the DCO**

Full details of the Developer Certificate of Origin can be found at [developercertificate.org](https://developercertificate.org/).

**The list of commits missing DCO signoff**:

- [sha](https://github.com///commits/sha) not signed off

<details>

Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).
</details>
`,
		},
		{
			name: "should add label and update status context if all commits are signed-off",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionOpen,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha", Commit: github.GitCommit{Message: "Signed-off-by: someone"}},
			},
			issueState: "open",
			hasDCONo:   false,
			hasDCOYes:  false,

			addedLabel:     fmt.Sprintf("/#3:%s", dcoYesLabel),
			expectedStatus: github.StatusSuccess,
		},
		{
			name: "should add label and update status context and remove old labels if all commits are signed-off",
			pullRequestEvent: scm.PullRequestHook{
				Action:      scm.ActionOpen,
				PullRequest: scm.PullRequest{Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha", Commit: github.GitCommit{Message: "Signed-off-by: someone"}},
			},
			issueState: "open",
			hasDCONo:   true,
			hasDCOYes:  false,

			addedLabel:     fmt.Sprintf("/#3:%s", dcoYesLabel),
			removedLabel:   fmt.Sprintf("/#3:%s", dcoNoLabel),
			expectedStatus: github.StatusSuccess,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakegithub.FakeClient{
				CombinedStatuses: make(map[string]*github.CombinedStatus),
				CreatedStatuses:  make(map[string][]github.Status),
				PullRequests:     map[int]*scm.PullRequest{tc.pullRequestEvent.PullRequest.Number: &tc.pullRequestEvent.PullRequest},
				IssueComments:    make(map[int][]scm.Comment),
				CommitMap: map[string][]scm.RepositorysitoryCommit{
					"/#3": tc.commits,
				},
			}
			if tc.hasDCOYes {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, fmt.Sprintf("/#3:%s", dcoYesLabel))
			}
			if tc.hasDCONo {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, fmt.Sprintf("/#3:%s", dcoNoLabel))
			}
			combinedStatus := &github.CombinedStatus{
				Statuses: []github.Status{},
			}
			if tc.status != "" {
				fc.CreatedStatuses["sha"] = []github.Status{
					{Context: dcoContextName, State: tc.status},
				}
				combinedStatus = &github.CombinedStatus{
					SHA: "sha",
					Statuses: []github.Status{
						{Context: dcoContextName, State: tc.status},
					},
				}
			}
			fc.CombinedStatuses["sha"] = combinedStatus

			if err := handlePullRequest(fc, &fakePruner{}, logrus.WithField("plugin", pluginName), tc.pullRequestEvent); err != nil {
				t.Errorf("For case %s, didn't expect error from dco plugin: %v", tc.name, err)
			}
			ok := tc.addedLabel == ""
			if !ok {
				for _, label := range fc.IssueLabelsAdded {
					if reflect.DeepEqual(tc.addedLabel, label) {
						ok = true
						break
					}
				}
			}
			if !ok {
				t.Errorf("Expected to add: %#v, Got %#v in case %s.", tc.addedLabel, fc.IssueLabelsAdded, tc.name)
			}
			ok = tc.removedLabel == ""
			if !ok {
				for _, label := range fc.IssueLabelsRemoved {
					if reflect.DeepEqual(tc.removedLabel, label) {
						ok = true
						break
					}
				}
			}
			if !ok {
				t.Errorf("Expected to remove: %#v, Got %#v in case %s.", tc.removedLabel, fc.IssueLabelsRemoved, tc.name)
			}

			// check status is set as expected
			statuses := fc.CreatedStatuses["sha"]
			if len(statuses) == 0 && tc.expectedStatus != "" {
				t.Errorf("Expected dco status to be %q, but it was not set", tc.expectedStatus)
			}
			found := false
			for _, s := range statuses {
				if s.Context == dcoContextName {
					found = true
					if s.State != tc.expectedStatus {
						t.Errorf("Expected dco status to be %q but it was %q", tc.expectedStatus, s.State)
					}
				}
			}
			if !found && tc.expectedStatus != "" {
				t.Errorf("Expect dco status to be %q, but it was not found", tc.expectedStatus)
			}

			comments := fc.IssueCommentsAdded
			if len(comments) == 0 && tc.addedComment != "" {
				t.Errorf("Expected comment with body %q to be added, but it was not", tc.addedComment)
				return
			}
			if len(comments) > 1 {
				t.Errorf("did not expect more than one comment to be created")
			}
			if len(comments) != 0 && comments[0] != tc.addedComment {
				t.Errorf("expected comment to be %q but it was %q", tc.addedComment, comments[0])
			}
		})
	}
}
func TestHandleComment(t *testing.T) {
	var testcases = []struct {
		// test settings
		name string

		// PR settings
		commentEvent github.GenericCommentEvent
		pullRequests map[int]*scm.PullRequest
		commits      []scm.RepositorysitoryCommit
		issueState   string
		hasDCOYes    bool
		hasDCONo     bool
		// status of the DCO github context
		status string

		// expectations
		addedLabel     string
		removedLabel   string
		expectedStatus string
		// org/repo#number:body
		addedComment string
		// org/repo#issuecommentid
		removedComment string
	}{
		{
			name: "should not do anything if comment does not match /check-dco",
			commentEvent: github.GenericCommentEvent{
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "not-the-trigger",
				IsPR:       true,
				Number:     3,
			},
			pullRequests: map[int]*scm.PullRequest{
				3: {Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
		},
		{
			name: "should add 'no' label & status context and add a comment if no commits have sign off",
			commentEvent: github.GenericCommentEvent{
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/check-dco",
				IsPR:       true,
				Number:     3,
			},
			pullRequests: map[int]*scm.PullRequest{
				3: {Number: 3, Head: scm.PullRequestBranch{SHA: "sha"}},
			},
			commits: []scm.RepositorysitoryCommit{
				{SHA: "sha", Commit: github.GitCommit{Message: "not a sign off"}},
			},
			issueState: "open",
			hasDCONo:   false,
			hasDCOYes:  false,

			addedLabel:     fmt.Sprintf("/#3:%s", dcoNoLabel),
			expectedStatus: github.StatusFailure,
			addedComment: `/#3:Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits.

:memo: **Please follow instructions in the [contributing guide](https://github.com///blob/master/CONTRIBUTING.md) to update your commits with the DCO**

Full details of the Developer Certificate of Origin can be found at [developercertificate.org](https://developercertificate.org/).

**The list of commits missing DCO signoff**:

- [sha](https://github.com///commits/sha) not a sign off

<details>

Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).
</details>
`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakegithub.FakeClient{
				CreatedStatuses:  make(map[string][]github.Status),
				CombinedStatuses: make(map[string]*github.CombinedStatus),
				PullRequests:     tc.pullRequests,
				IssueComments:    make(map[int][]scm.Comment),
				CommitMap: map[string][]scm.RepositorysitoryCommit{
					"/#3": tc.commits,
				},
			}
			if tc.hasDCOYes {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, fmt.Sprintf("/#3:%s", dcoYesLabel))
			}
			if tc.hasDCONo {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, fmt.Sprintf("/#3:%s", dcoNoLabel))
			}
			combinedStatus := &github.CombinedStatus{
				Statuses: []github.Status{},
			}
			if tc.status != "" {
				fc.CreatedStatuses["sha"] = []github.Status{
					{Context: dcoContextName, State: tc.status},
				}
				combinedStatus = &github.CombinedStatus{
					SHA: "sha",
					Statuses: []github.Status{
						{Context: dcoContextName, State: tc.status},
					},
				}
			}
			fc.CombinedStatuses["sha"] = combinedStatus

			if err := handleComment(fc, &fakePruner{}, logrus.WithField("plugin", pluginName), tc.commentEvent); err != nil {
				t.Errorf("For case %s, didn't expect error from dco plugin: %v", tc.name, err)
			}
			ok := tc.addedLabel == ""
			if !ok {
				for _, label := range fc.IssueLabelsAdded {
					if reflect.DeepEqual(tc.addedLabel, label) {
						ok = true
						break
					}
				}
			}
			if !ok {
				t.Errorf("Expected to add: %#v, Got %#v in case %s.", tc.addedLabel, fc.IssueLabelsAdded, tc.name)
			}
			ok = tc.removedLabel == ""
			if !ok {
				for _, label := range fc.IssueLabelsRemoved {
					if reflect.DeepEqual(tc.removedLabel, label) {
						ok = true
						break
					}
				}
			}
			if !ok {
				t.Errorf("Expected to remove: %#v, Got %#v in case %s.", tc.removedLabel, fc.IssueLabelsRemoved, tc.name)
			}

			// check status is set as expected
			statuses := fc.CreatedStatuses["sha"]
			if len(statuses) == 0 && tc.expectedStatus != "" {
				t.Errorf("Expected dco status to be %q, but it was not set", tc.expectedStatus)
			}
			found := false
			for _, s := range statuses {
				if s.Context == dcoContextName {
					found = true
					if s.State != tc.expectedStatus {
						t.Errorf("Expected dco status to be %q but it was %q", tc.expectedStatus, s.State)
					}
				}
			}
			if !found && tc.expectedStatus != "" {
				t.Errorf("Expect dco status to be %q, but it was not found", tc.expectedStatus)
			}

			comments := fc.IssueCommentsAdded
			if len(comments) == 0 && tc.addedComment != "" {
				t.Errorf("Expected comment with body %q to be added, but it was not", tc.addedComment)
				return
			}
			if len(comments) > 1 {
				t.Errorf("did not expect more than one comment to be created")
			}
			if len(comments) != 0 && comments[0] != tc.addedComment {
				t.Errorf("expected comment to be %q but it was %q", tc.addedComment, comments[0])
			}
		})
	}
}

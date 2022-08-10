package repo

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gogs/git-module"
	api "github.com/gogs/go-gogs-client"
	"gogs.io/gogs/internal/context"
	"gogs.io/gogs/internal/db"
	"gogs.io/gogs/internal/gitutil"
	log "unknwon.dev/clog/v2"
)

func ListPullRequests(c *context.APIContext, form api.ListPullRequestsOptions) {
	prs, maxResults, err := db.PullRequests(c.Repo.Repository.ID, &db.PullRequestsOptions{
		Page:        c.QueryInt("page"),
		State:       c.QueryTrim("state"),
		SortType:    c.QueryTrim("sort"),
		Labels:      c.QueryStrings("labels"),
		MilestoneID: c.QueryInt64("milestone"),
	})
	if err != nil {
		c.Error(err, "get pull requests")
		return
	}

	apiPrs := make([]*api.PullRequest, len(prs))
	for i := range prs {
		prs[i].LoadIssue()
		prs[i].LoadAttributes()
		prs[i].GetBaseRepo()
		prs[i].GetHeadRepo()
		apiPrs[i] = prs[i].APIFormat()
	}
	c.SetLinkHeader(int(maxResults), db.ItemsPerPage)
	c.JSONSuccess(&apiPrs)
}

func GetPullRequest(c *context.APIContext) {
	pr, err := db.GetPullRequestByIndex(c.Repo.Repository.ID, c.ParamsInt64(":index"))
	if err != nil {
		c.NotFoundOrError(err, "get pull")
		return
	}

	pr.GetBaseRepo()
	pr.GetHeadRepo()

	c.JSONSuccess(pr.APIFormat())
}

// CreatePullRequest does what it says
func CreatePullRequest(ctx *context.APIContext, form api.CreatePullRequestOption) {
	var (
		repo        = ctx.Repo.Repository
		labelIDs    []int64
		assigneeID  int64
		milestoneID int64
	)

	// Get repo/branch information
	headUser, headRepo, headGitRepo, prInfo, baseBranch, headBranch := parseCompareInfo(ctx, form)
	if ctx.Written() {
		return
	}

	// Check if another PR exists with the same targets
	existingPr, err := db.GetUnmergedPullRequest(headRepo.ID, ctx.Repo.Repository.ID, headBranch, baseBranch)
	if err != nil {
		if !db.IsErrPullRequestNotExist(err) {
			ctx.Error(err, "GetUnmergedPullRequest")
			return
		}
	} else {
		err = db.ErrPullRequestAlreadyExists{
			ID:         existingPr.ID,
			IssueID:    existingPr.Index,
			HeadRepoID: existingPr.HeadRepoID,
			BaseRepoID: existingPr.BaseRepoID,
			HeadBranch: existingPr.HeadBranch,
			BaseBranch: existingPr.BaseBranch,
		}
		ctx.Error(err, "GetUnmergedPullRequest")
		return
	}

	if len(form.Labels) > 0 {
		labels, err := db.GetLabelsInRepoByIDs(ctx.Repo.Repository.ID, form.Labels)
		if err != nil {
			ctx.Error(err, "GetLabelsInRepoByIDs")
			return
		}

		labelIDs = make([]int64, len(labels))
		for i := range labels {
			labelIDs[i] = labels[i].ID
		}
	}

	if form.Milestone > 0 {
		milestone, err := db.GetMilestoneByRepoID(ctx.Repo.Repository.ID, milestoneID)
		if err != nil {
			if db.IsErrMilestoneNotExist(err) {
				ctx.Status(404)
			} else {
				ctx.Error(err, "GetMilestoneByRepoID")
			}
			return
		}

		milestoneID = milestone.ID
	}

	if len(form.Assignee) > 0 {
		assigneeUser, err := db.GetUserByName(form.Assignee)
		if err != nil {
			if db.IsErrUserNotExist(err) {
				ctx.Error(fmt.Errorf("assignee does not exist: [name: %s]", form.Assignee), "GetUserByName")
			} else {
				ctx.Error(err, "GetUserByName")
			}
			return
		}

		assignee, err := repo.GetAssigneeByID(assigneeUser.ID)
		if err != nil {
			ctx.Error(err, "GetAssigneeByID")
			return
		}

		assigneeID = assignee.ID
	}

	patch, err := headGitRepo.DiffBinary(prInfo.MergeBase, headBranch)
	if err != nil {
		ctx.Error(err, "get patch")
		return
	}

	prIssue := &db.Issue{
		RepoID:      repo.ID,
		Index:       repo.NextIssueIndex(),
		Title:       form.Title,
		PosterID:    ctx.User.ID,
		Poster:      ctx.User,
		MilestoneID: milestoneID,
		AssigneeID:  assigneeID,
		IsPull:      true,
		Content:     form.Body,
	}
	pr := &db.PullRequest{
		HeadRepoID:   headRepo.ID,
		BaseRepoID:   repo.ID,
		HeadUserName: headUser.Name,
		HeadBranch:   headBranch,
		BaseBranch:   baseBranch,
		HeadRepo:     headRepo,
		BaseRepo:     repo,
		MergeBase:    prInfo.MergeBase,
		Type:         db.PULL_REQUEST_GOGS,
	}

	if err := db.NewPullRequest(repo, prIssue, labelIDs, []string{}, pr, patch); err != nil {
		ctx.Error(err, "NewPullRequest")
		return
	} else if err := pr.PushToBaseRepo(); err != nil {
		ctx.Error(err, "PushToBaseRepo")
		return
	}

	log.Trace("Pull request created: %d/%d", repo.ID, prIssue.ID)
	ctx.JSON(201, pr.APIFormat())
}

// EditPullRequest does what it says
func EditPullRequest(ctx *context.APIContext, form api.EditPullRequestOption) {
	log.Info("请求过来了")
	pr, err := db.GetPullRequestByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if db.IsErrPullRequestNotExist(err) {
			ctx.Status(404)
		} else {
			ctx.Error(err, "GetPullRequestByIndex")
		}
		return
	}

	pr.LoadIssue()
	issue := pr.Issue

	if !issue.IsPoster(ctx.User.ID) && !ctx.Repo.IsWriter() {
		ctx.Status(403)
		return
	}

	if len(form.Title) > 0 {
		issue.Title = form.Title
	}
	if len(form.Body) > 0 {
		issue.Content = form.Body
	}

	if ctx.Repo.IsWriter() && len(form.Assignee) > 0 &&
		(issue.Assignee == nil || issue.Assignee.LowerName != strings.ToLower(form.Assignee)) {
		if len(form.Assignee) == 0 {
			issue.AssigneeID = 0
		} else {
			assignee, err := db.GetUserByName(form.Assignee)
			if err != nil {
				if db.IsErrUserNotExist(err) {
					ctx.Error(fmt.Errorf("assignee does not exist: [name: %s]", form.Assignee), "GetUserByName")
				} else {
					ctx.Error(err, "GetUserByName")
				}
				return
			}
			issue.AssigneeID = assignee.ID
		}

		if err = db.UpdateIssueUserByAssignee(issue); err != nil {
			ctx.Error(err, "UpdateIssueUserByAssignee")
			return
		}
	}
	if ctx.Repo.IsWriter() && form.Milestone != 0 &&
		issue.MilestoneID != form.Milestone {
		oldMilestoneID := issue.MilestoneID
		issue.MilestoneID = form.Milestone
		if err = db.ChangeMilestoneAssign(ctx.User, issue, oldMilestoneID); err != nil {
			ctx.Error(err, "ChangeMilestoneAssign")
			return
		}
	}

	if err = db.UpdateIssue(issue); err != nil {
		ctx.Error(err, "UpdateIssue")
		return
	}
	if form.State != nil {
		if err = issue.ChangeStatus(ctx.User, ctx.Repo.Repository, api.STATE_CLOSED == api.StateType(*form.State)); err != nil {
			ctx.Error(err, "ChangeStatus")
			return
		}
	}

	// Refetch from database
	pr, err = db.GetPullRequestByIndex(ctx.Repo.Repository.ID, pr.Index)
	if err != nil {
		if db.IsErrPullRequestNotExist(err) {
			ctx.Status(404)
		} else {
			ctx.Error(err, "GetPullRequestByIndex")
		}
		return
	}

	ctx.JSON(201, pr.APIFormat())
}

// IsPullRequestMerged checks if a PR exists given an index
//  - Returns 204 if it exists
//    Otherwise 404
func IsPullRequestMerged(ctx *context.APIContext) {
	pr, err := db.GetPullRequestByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if db.IsErrPullRequestNotExist(err) {
			ctx.Status(404)
		} else {
			ctx.Error(err, "GetPullRequestByIndex")
		}
		return
	}

	if pr.HasMerged {
		ctx.Status(204)
	}
	ctx.Status(404)
}

// MergePullRequest merges a PR given an index
func MergePullRequest(ctx *context.APIContext) {
	pr, err := db.GetPullRequestByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if db.IsErrPullRequestNotExist(err) {
			ctx.Error(err, "GetPullRequestByIndex")
		} else {
			ctx.Error(err, "GetPullRequestByIndex")
		}
		return
	}

	if err = pr.GetHeadRepo(); err != nil {
		ctx.Error(err, "GetHeadRepo")
		return
	}

	pr.LoadIssue()
	pr.Issue.Repo = ctx.Repo.Repository

	if ctx.IsTokenAuth {
		// Update issue-user.
		if err = pr.Issue.ReadBy(ctx.User.ID); err != nil {
			ctx.Error(err, "ReadBy")
			return
		}
	}

	if pr.Issue.IsClosed {
		ctx.Status(404)
		return
	}

	if !pr.CanAutoMerge() || pr.HasMerged {
		ctx.Status(405)
		return
	}
	// TODO
	merge_style := ctx.Query("merge_style")
	if merge_style == "" {
		merge_style = "create_merge_commit"
	}
	commit_description := ctx.Query("commit_description")
	log.Info("%s MergePullRequest %s %s", ctx.Repo.Repository.RepoPath(), merge_style, commit_description)
	spew.Dump(ctx.Repo)
	if ctx.Repo.GitRepo == nil {
		ctx.Repo.GitRepo, _ = git.Open(ctx.Repo.Repository.RepoPath())
	}
	if err := pr.Merge(ctx.User, ctx.Repo.GitRepo, db.MergeStyle(merge_style), commit_description); err != nil {
		ctx.Error(err, "Merge")
		return
	}

	log.Trace("Pull request merged: %d", pr.ID)
	ctx.Status(200)
}

func parseCompareInfo(ctx *context.APIContext, form api.CreatePullRequestOption) (*db.User, *db.Repository, *git.Repository, *gitutil.PullRequestMeta, string, string) {
	baseRepo := ctx.Repo.Repository

	// Get compared branches information
	// format: <base branch>...[<head repo>:]<head branch>
	// base<-head: master...head:feature
	// same repo: master...feature

	baseBranch := form.Base

	var (
		headUser   *db.User
		headBranch string
		isSameRepo bool
		err        error
	)

	// If there is no head repository, it means pull request between same repository.
	headInfos := strings.Split(form.Head, ":")
	if len(headInfos) == 1 {
		isSameRepo = true
		headUser = ctx.Repo.Owner
		headBranch = headInfos[0]

	} else if len(headInfos) == 2 {
		headUser, err = db.GetUserByName(headInfos[0])
		if err != nil {
			ctx.NotFoundOrError(err, "get user by name")
			return nil, nil, nil, nil, "", ""
		}
		headBranch = headInfos[1]
		isSameRepo = headUser.ID == baseRepo.OwnerID
	} else {
		ctx.NotFound()
		return nil, nil, nil, nil, "", ""
	}
	log.Info("Base branch: %s", baseBranch)
	// Check if base branch is valid.
	spew.Dump(ctx.Repo.Repository)
	// if ctx.Repo.GitRepo.HasBranch(baseBranch) {
	if !git.RepoHasBranch(ctx.Repo.Repository.RepoPath(), baseBranch) {
		ctx.NotFound()
		return nil, nil, nil, nil, "", ""
	}

	var (
		headRepo    *db.Repository
		headGitRepo *git.Repository
	)

	// In case user included redundant head user name for comparison in same repository,
	// no need to check the fork relation.
	if !isSameRepo {
		var has bool
		headRepo, has, err = db.HasForkedRepo(headUser.ID, baseRepo.ID)
		if err != nil {
			ctx.Error(err, "get forked repository")
			return nil, nil, nil, nil, "", ""
		} else if !has {
			log.Trace("ParseCompareInfo [base_repo_id: %d]: does not have fork or in same repository", baseRepo.ID)
			ctx.NotFound()
			return nil, nil, nil, nil, "", ""
		}

		headGitRepo, err = git.Open(db.RepoPath(headUser.Name, headRepo.Name))
		if err != nil {
			ctx.Error(err, "open repository")
			return nil, nil, nil, nil, "", ""
		}
	} else {
		headRepo = ctx.Repo.Repository
		if ctx.Repo.GitRepo != nil {
			headGitRepo = ctx.Repo.GitRepo
		} else {
			headGitRepo, err = git.Open(db.RepoPath(headUser.Name, headRepo.Name))
			if err != nil {
				ctx.Error(err, "open repository")
				return nil, nil, nil, nil, "", ""
			}
		}

	}

	if !ctx.User.IsWriterOfRepo(headRepo) && !ctx.User.IsAdmin {
		log.Trace("ParseCompareInfo [base_repo_id: %d]: does not have write access or site admin", baseRepo.ID)
		ctx.NotFound()
		return nil, nil, nil, nil, "", ""
	}

	// Check if head branch is valid.
	// if !headGitRepo.HasBranch(headBranch) {
	if !git.RepoHasBranch(ctx.Repo.Repository.RepoPath(), headBranch) {
		ctx.NotFound()
		return nil, nil, nil, nil, "", ""
	}

	baseRepoPath := db.RepoPath(baseRepo.Owner.Name, baseRepo.Name)
	spew.Dump(headGitRepo)
	prInfo, err := gitutil.Module.PullRequestMeta(baseRepoPath, baseRepoPath, headBranch, baseBranch)
	if err != nil {
		ctx.Error(err, "get pull request meta")
		return nil, nil, nil, nil, "", ""
	}
	return headUser, headRepo, headGitRepo, prInfo, baseBranch, headBranch
}

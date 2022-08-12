// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"fmt"
	"net/http"

	"github.com/gogs/git-module"
	api "github.com/gogs/go-gogs-client"
	log "unknwon.dev/clog/v2"

	"gogs.io/gogs/internal/context"
	"gogs.io/gogs/internal/db"
	"gogs.io/gogs/internal/route/api/v1/convert"
)

// https://github.com/gogs/go-gogs-client/wiki/Repositories#get-branch
func GetBranch(c *context.APIContext) {
	branch, err := c.Repo.Repository.GetBranch(c.Params("*"))
	if err != nil {
		c.NotFoundOrError(err, "get branch")
		return
	}

	commit, err := branch.GetCommit()
	if err != nil {
		c.Error(err, "get commit")
		return
	}

	c.JSONSuccess(convert.ToBranch(branch, commit))
}

// https://github.com/gogs/go-gogs-client/wiki/Repositories#list-branches
func ListBranches(c *context.APIContext) {
	branches, err := c.Repo.Repository.GetBranches()
	if err != nil {
		c.Error(err, "get branches")
		return
	}

	apiBranches := make([]*api.Branch, len(branches))
	for i := range branches {
		commit, err := branches[i].GetCommit()
		if err != nil {
			c.Error(err, "get commit")
			return
		}
		apiBranches[i] = convert.ToBranch(branches[i], commit)
	}

	c.JSONSuccess(&apiBranches)
}

func ListProtectionsBranches(c *context.APIContext) {
	branches, err := db.GetProtectBranchesByRepoID(c.Repo.Repository.ID)
	if err != nil {
		c.Error(err, "get Protect branches")
		return
	}

	c.JSONSuccess(&branches)
}

// DeleteBranch get a branch of a repository
func DeleteBranch(ctx *context.APIContext) {
	branchName := ctx.Params("*")

	_, err := ctx.Repo.Repository.GetBranch(branchName)
	if err != nil {
		ctx.NotFoundOrError(err, "get branch")
		return
	}

	if ctx.Repo.Repository.DefaultBranch == branchName {
		ctx.Error(fmt.Errorf("can not delete default branch"), "DefaultBranch")
		return
	}

	isProtected, err := ctx.Repo.Repository.IsProtectedBranch(branchName)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if isProtected {
		ctx.Error(fmt.Errorf("branch protected"), "IsProtectedBranch")
		return
	}
	repoPath := db.RepoPath(ctx.Repo.Owner.Name, ctx.Repo.Repository.Name)
	ctx.Repo.GitRepo, err = git.Open(repoPath)
	if err != nil {
		ctx.Error(err, "open repository")
		return
	}
	if err := ctx.Repo.GitRepo.DeleteBranch(branchName, git.DeleteBranchOptions{
		Force: true,
	}); err != nil {
		ctx.Error(err, "DeleteBranch")
		return
	}

	if err := db.PrepareWebhooks(ctx.Repo.Repository, db.HOOK_EVENT_DELETE, &api.DeletePayload{
		Ref:        branchName,
		RefType:    "branch",
		PusherType: api.PUSHER_TYPE_USER,
		Repo:       ctx.Repo.Repository.APIFormatLegacy(nil),
		Sender:     ctx.User.APIFormat(),
	}); err != nil {
		log.Error("Failed to prepare webhooks for %q: %v", db.HOOK_EVENT_DELETE, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

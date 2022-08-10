// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package db

func (repo *Repository) GetPullRequest(name string) (*PullRequest, error) {
	return &PullRequest{}, nil
}

func (repo *Repository) GetPullRequests() ([]*PullRequest, error) {
	return nil, nil
}

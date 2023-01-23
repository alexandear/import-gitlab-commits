# Import GitLab Commits

The tool to import commits from private GitLab to separate repo. Can be used to show your programming activity for another company in GitHub.

# Getting Started

1. Download and install [Go 1.19](https://go.dev/dl/).
2. Install the program by running the command in a shell:
```shell
go install github.com/alexandear/import-gitlab-commits@latest
```

3. Run `import-gitlab-commits`:
```shell
export GITLAB_BASE_URL=https://gitlab.yourcompany.com
export GITLAB_TOKEN=yourgitlabtoken
export COMMITTER_NAME="John Doe"
export COMMITTER_EMAIL=john.doe@yourcompany.com

import-gitlab-commits
```

Contributions before running `import-gitlab-commits`:

<img src="./screenshots/contribs_before.png" width="1000">

After:

<img src="./screenshots/contribs_after.png" width="1000">

What work the tool does:
* gets current user info by `GITLAB_TOKEN`;
* fetches from `GITLAB_BASE_URL` projects that the current user contributed to;
* for all projects fetches commits where author's email is the current user's email;
* creates new repo `repo.gitlab.yourcompany.com.currentusername` and commits all fetched commits with message
`Project: GITLAB_PROJECT_ID commit: GITLAB_COMMIT_HASH` and commit date `GITLAB_COMMIT_DATE`.

To show the changes on GitHub you need to:
* create a new repo `yourcompany-contributions` in GitHub;
* open folder `repo.gitlab.yourcompany.com.currentusername`;
* add remote url `git remote add origin git@github.com:username/yourcompany-contributions.git`;
* push changes.

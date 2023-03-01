# Import GitLab Commits

The tool to import commits from private GitLab to separate repo. Can be used to show your programming activity for another company in GitHub.

# Getting Started

1. Download and install [Go 1.19](https://go.dev/dl/).
2. Install the program by running the command in a shell:
```shell
go install github.com/alexandear/import-gitlab-commits@latest
```

3. Set environment variables and run `import-gitlab-commits`:
```shell
export GITLAB_BASE_URL=<your_gitlab_server_url>
export GITLAB_TOKEN=<your_gitlab_token>
export COMMITTER_NAME="<Name Surname>"
export COMMITTER_EMAIL=<mail@example.com>

import-gitlab-commits
```

where
- `GITLAB_BASE_URL` is a GitLab [instance URL](https://stackoverflow.com/questions/58236175/what-is-a-gitlab-instance-url-and-how-can-i-get-it), e.g. `https://gitlab.com`, `https://gitlab.gnome.org` or any GitLab server;
- `GITLAB_TOKEN` is a personal [access token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html#create-a-personal-access-token);
- `COMMITTER_NAME` is a name that can be passed to `git config user.name`, e.g. `John Doe`;
- `COMMITTER_EMAIL` is an email valid for `git config user.email`, e.g. `john.doe@example.com`.

## Example

Contributions before running `import-gitlab-commits`:

<img src="./screenshots/contribs_before.png" width="1000">

After:

<img src="./screenshots/contribs_after.png" width="1000">

# Internals

What work the tool does:
* gets current user info by `GITLAB_TOKEN`;
* fetches from `GITLAB_BASE_URL` projects that the current user contributed to;
* for all projects fetches commits where author's email is the current user's email;
* creates new repo `repo.gitlab.yourcompany.com.currentusername` and commits all fetched commits with message
`Project: GITLAB_PROJECT_ID commit: GITLAB_COMMIT_HASH`, commit date `GITLAB_COMMIT_DATE`, and commit author `COMMITTER_NAME <COMMITTER_EMAIL>`.

To show the changes on GitHub you need to:
* create a new repo `yourcompany-contributions` in GitHub;
* open folder `repo.gitlab.yourcompany.com.currentusername`;
* add remote url `git remote add origin git@github.com:username/yourcompany-contributions.git`;
* push changes.

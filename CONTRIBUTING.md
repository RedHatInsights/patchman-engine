### How to Contribute

1. **Fork** the repository.
2. Clone your fork locally.
3. Make your changes in a new branch.
4. **Test** your changes thoroughly.
5. Open a **pull request** with a clear description of your changes.

### Guidelines

- Please follow the coding style guidelines.
- Ensure that all tests pass before submitting a pull request.
- Add test for new functionality

### Commits & messages

- Do not use semantic commit messages (chore, fix, feat, ...)
- Reference Jira task at the beginning of message: e.g. `RHINENG-21424: handle null/notnull filters`
- Break one big commit into multiple small commits (one per logical change)
- Put non-functional changes (e.g. refactor to extract a helper) into separate commits (or PRs)
- Write clear commit messages, a bit of "what" and a lot of "why"
- Don't squash commits

### Github PRs

- Reference Jira task at the beginning of PR name (e.g. `RHINENG-21424: add filter[severity]=null`)
- Multiple small PRs are better than one big PR
- Multiple commits are easier to review than one big commit
- PRs are rebased not merged - it's easier to follow a single stream of changes
- Merged PRs are automatically tagged with new version

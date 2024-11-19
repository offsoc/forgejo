---

name: "Pull Request Template"
about: "Template for all Pull Requests"
labels:

- test/needed

---

## Checklist

The [contributor guide](https://forgejo.org/docs/next/contributor/) contains information that will be helpful to first time contributors. There also are a few [conditions for merging Pull Requests in Forgejo repositories](https://codeberg.org/forgejo/governance/src/branch/main/PullRequestsAgreement.md). You are also welcome to join the [Forgejo development chatroom](https://matrix.to/#/#forgejo-development:matrix.org).

### Tests

- I added test coverage for Go changes...
  - [ ] in their respective `*_test.go` for unit tests.
  - [ ] in the `tests/integration` directory if it involves interactions with a live Forgejo server.
- I added test coverage for JavaScript changes...
  - [ ] in `web_src/js/*.test.js` if it can be unit tested.
  - [ ] in `tests/e2e/*.test.e2e.js` if it requires interactions with a live Forgejo server (see also the [developer guide for JavaScript testing](https://codeberg.org/forgejo/forgejo/src/branch/forgejo/tests/e2e/README.md#end-to-end-tests)).

### Documentation

- [ ] I created a pull request [to the documentation](https://codeberg.org/forgejo/docs) to explain to Forgejo users how to use this change.
- [ ] I did not document these changes and I do not expect someone else to do it.

### Release notes

- [ ] I do not want this change to show in the release notes.
- [ ] I want the title to show in the release notes with a link to this pull request.
- [ ] I want the content of the `release-notes/<pull request number>.md` to be be used for the release notes instead of the title.

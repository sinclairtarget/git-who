# git-who
`git-who` tallies authorship. Because sometimes you want to know who is
responsible not just for a block of code but for an entire project or feature.

You can think of `git-who` as `git blame` but for file trees, i.e. directories
and their contents. Or maybe as `git shortlog -n` with additional features.

A work in progress.

## Test Repository Submodule
Some of the automated tests for `git-who` need to run against a Git repository.
A test repository is attached to this repository as a submodule.

If you want to run the automated tests, you will first need to set up the
submodule:

```
$ git submodule update --init
```

# DEVELOPMENT
## Test Repository Submodule
Some of the automated tests for `git-who` need to run against a Git repository.
Test repositories are attached to this repository as submodules.

If you want to run the automated tests, you will first need to set up the
submodules:

```
$ git submodule update --init
```

## Automated Tests
The unit and integration tests, written in Go, can be run using:

```
$ rake test
```

## Functional Tests
There are some end-to-end/functional tests written in Ruby. These require the
`minitest` gem. You can run them using:

```
$ rake test:functional
```

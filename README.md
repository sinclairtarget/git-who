# git-who
 ![Vanity screenshot](./screenshots/vanity.png)

`git-who` is a command-line tool for answering that age-old question:

> _Who wrote this code?!_

Unlike `git blame`, which can tell you who wrote a _line_ of code, `git-who`
can help you identify the people responsible for entire components or
subsystems in a codebase. You can think of `git-who` sort of like `git-blame`
but for file trees rather than individual files.

## Installation
TBD.

## Usage
_(In the following examples, `git-who` is invoked as `git who`, which requires
setting up a Git alias. See the [Git Alias](#git-alias) section below.)_

`git who` has three subcommands. Each subcommand gives you a different view of
authorship in your Git repository.

### The `table` Subcommand
The `table` subcommand is the default subcommand. You can invoke it explicitly
as `git who table` or implicitly just as `git who`.

The `table` subcommand prints a table summarizing the contributions of every
author who has made commits in the repository.

```
~/repos/cpython$ git who
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  2 mon. ago    11213│
│Victor Stinner                    1 week ago     7193│
│Fred Drake                        13 yr. ago     5465│
│Georg Brandl                      1 year ago     5294│
│Benjamin Peterson                 4 mon. ago     4724│
│Raymond Hettinger                 1 month ago    4235│
│Serhiy Storchaka                  1 day ago      3366│
│Antoine Pitrou                    10 mon. ago    3180│
│Jack Jansen                       18 yr. ago     2978│
│Martin v. Löwis                   9 yr. ago      2690│
│...3026 more...                                      │
└─────────────────────────────────────────────────────┘
```

You can specify a path to filter the results to only commits that
touched files under the given path:
```
~/repos/cpython$ git who Tools/
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  8 mon. ago      820│
│Barry Warsaw                      1 year ago      279│
│Martin v. Löwis                   9 yr. ago       242│
│Victor Stinner                    1 month ago     235│
│Steve Dower                       1 month ago     228│
│Jeremy Hylton                     19 yr. ago      178│
│Mark Shannon                      4 hr. ago       131│
│Serhiy Storchaka                  2 mon. ago      118│
│Erlend E. Aasland                 1 week ago      117│
│Christian Heimes                  2 yr. ago       114│
│...267 more...                                       │
└─────────────────────────────────────────────────────┘
```

You can also specify a branch name, tag name, or any "commit-ish" to
filter the results to commits reachable from the specified commit:
```
~/repos/cpython$ git who v3.7.1
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  6 yr. ago     10986│
│Fred Drake                        13 yr. ago     5465│
│Georg Brandl                      8 yr. ago      5291│
│Benjamin Peterson                 6 yr. ago      4599│
│Victor Stinner                    6 yr. ago      4462│
│Raymond Hettinger                 6 yr. ago      3667│
│Antoine Pitrou                    6 yr. ago      3149│
│Jack Jansen                       18 yr. ago     2978│
│Martin v. Löwis                   9 yr. ago      2690│
│Tim Peters                        10 yr. ago     2489│
│...550 more...                                       │
└─────────────────────────────────────────────────────┘
```

Just like with `git` itself, when there is ambiguity between a path name
and a commit-ish, you can use `--` to clarfiy the distinction. The
following command will show you contributions to the file or directory
called `foo` even if there is also a branch called `foo` in your repository:
```
$ git who -- foo
```

#### Options
The `-m`, `-l`, and `-f` flags allow you to sort the table by different
metrics.

The `-m` flag sorts the table by the "Last Edit" column, showing who
edited the repository most recently.

The `-l` flag sorts the table by number of lines modified, adding some more
columns:

```
$ git who -l
┌──────────────────────────────────────────────────────────────────────────────┐
│Author                           Last Edit   Commits   Files       Lines (+/-)│
├──────────────────────────────────────────────────────────────────────────────┤
│Jiang Xin                        8 mon. ago      331     471  253283 /  226023│
│Junio C Hamano                   19 hr. ago     7993    2492  247398 /  105550│
│Peter Krefting                   3 mon. ago      122     564  160447 /  135430│
│Tran Ngoc Quan                   2 yr. ago        84     296  148753 /  126581│
│Alexander Shopov                 1 week ago       82     298  143828 /  114936│
│Jordi Mas                        2 mon. ago       59     222  110952 /   95889│
│Jean-Noël Avila                  1 month ago     126     383   96875 /   90328│
│Ralf Thielow                     2 mon. ago      166     470   88881 /   64093│
│Dimitriy Ryazantcev              1 year ago       30     279   84741 /   62953│
│Jeff King                        3 days ago     4421    2056   86235 /   47464│
│...2243 more...                                                               │
└──────────────────────────────────────────────────────────────────────────────┘
```

The `-f` flag sorts the table by the number of files modified.

There is also a `--csv` option that outputs the table as a CSV file to stdout.

Run `git-who table --help` to see additional options for the `table` subcommand.

### The `tree` Subcommand
TODO.

### The `hist` Subcommand
TODO.

## Git Alias
You can invoke `git-who` as `git who` by setting up an alias in your global Git
config:

```
[alias]
    who = "!git-who"
```

See [here](https://git-scm.com/book/en/v2/Git-Basics-Git-Aliases) for more
information about Git aliases.

## DEVELOPMENT
### Test Repository Submodule
Some of the automated tests for `git-who` need to run against a Git repository.
A test repository is attached to this repository as a submodule.

If you want to run the automated tests, you will first need to set up the
submodule:

```
$ git submodule update --init
```

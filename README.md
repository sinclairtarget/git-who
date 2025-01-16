# git-who
 ![Vanity screenshot](./screenshots/vanity.png)

`git-who` is a command-line tool for answering that age-old question:

> _Who wrote this code?!_

Unlike `git blame`, which can tell you who wrote a _line_ of code, `git-who`
is meant to help you identify the people responsible for entire components or
subsystems in a codebase. You can think of `git-who` sort of like `git-blame`
but for file trees rather than individual files.

## Installation
### Precompiled Binaries
See [releases](https://github.com/sinclairtarget/git-who/releases).

### From Source
This requires that you have Go, Ruby, and the `rake` Ruby gem installed.

```
$ git clone git@github.com:sinclairtarget/git-who.git
$ cd git-who
$ rake
$ ./git-who --version
```

## Usage
_(In the following examples, `git-who` is invoked as `git who`, which requires
setting up a Git alias. See the [Git Alias](#git-alias) section below.)_

`git who` has three subcommands. Each subcommand gives you a different view of
authorship in your Git repository.

### The `table` Subcommand
The `table` subcommand is the default subcommand. You can invoke it explicitly
as `git who table` or implicitly just as `git who`.

The `table` subcommand prints a table summarizing the contributions of every
author who has made commits in the repository:

```
~/clones/cpython$ git who
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  2 mon. ago   11,213│
│Victor Stinner                    1 week ago    7,193│
│Fred Drake                        13 yr. ago    5,465│
│Georg Brandl                      1 year ago    5,294│
│Benjamin Peterson                 4 mon. ago    4,724│
│Raymond Hettinger                 1 month ago   4,235│
│Serhiy Storchaka                  3 days ago    3,366│
│Antoine Pitrou                    10 mon. ago   3,180│
│Jack Jansen                       18 yr. ago    2,978│
│Martin v. Löwis                   9 yr. ago     2,690│
│...3,026 more...                                     │
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
~/clones/cpython$ git who v3.7.1
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  6 yr. ago    10,986│
│Fred Drake                        13 yr. ago    5,465│
│Georg Brandl                      8 yr. ago     5,291│
│Benjamin Peterson                 6 yr. ago     4,599│
│Victor Stinner                    6 yr. ago     4,462│
│Raymond Hettinger                 6 yr. ago     3,667│
│Antoine Pitrou                    6 yr. ago     3,149│
│Jack Jansen                       18 yr. ago    2,978│
│Martin v. Löwis                   9 yr. ago     2,690│
│Tim Peters                        10 yr. ago    2,489│
│...550 more...                                       │
└─────────────────────────────────────────────────────┘
```

Revision ranges also work. This shows the commits made after the release
of 3.10.9 up to the release of 3.11.9:
```
~/clones/cpython$ git who v3.10.9..v3.11.9
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Miss Islington (bot)              9 mon. ago    2,551│
│Victor Stinner                    9 mon. ago      367│
│Serhiy Storchaka                  9 mon. ago      304│
│Erlend Egeberg Aasland            2 yr. ago       202│
│Christian Heimes                  2 yr. ago       200│
│Mark Shannon                      1 year ago      157│
│Irit Katriel                      10 mon. ago     135│
│Nikita Sobolev                    10 mon. ago     126│
│Pablo Galindo Salgado             1 year ago      117│
│Pablo Galindo                     9 mon. ago       97│
│...574 more...                                       │
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
│Author                          Last Edit   Commits   Files        Lines (+/-)│
├──────────────────────────────────────────────────────────────────────────────┤
│Guido van Rossum                2 mon. ago   11,213  14,135     1.3m / 793,252│
│Antoine Pitrou                  10 mon. ago   3,180   3,868  944,685 / 776,587│
│Jack Jansen                     18 yr. ago    2,978   5,887  836,527 / 691,078│
│Benjamin Peterson               4 mon. ago    4,724   6,957  690,740 / 781,700│
│Georg Brandl                    1 year ago    5,294   9,139  644,620 / 640,217│
│Martin v. Löwis                 9 yr. ago     2,690   4,557  570,632 / 389,794│
│Victor Stinner                  1 week ago    7,193  11,382  464,474 / 460,396│
│Brett Cannon                    1 month ago   2,022   2,841  305,631 / 283,178│
│Serhiy Storchaka                3 days ago    3,366   9,955  335,209 / 208,899│
│Christian Heimes                1 year ago    1,553   4,191  339,706 / 178,947│
│...3,022 more...                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

The `-f` flag sorts the table by the number of files modified.

There is also a `--csv` option that outputs the table as a CSV file to stdout.

Run `git-who table --help` to see additional options for the `table` subcommand.

### The `tree` Subcommand
The `tree` subcommand prints out a file tree showing files in the working tree
just like [tree](https://en.wikipedia.org/wiki/Tree_(command)). Each node in the
file tree is annotated with information showing which author contributed the most
to files at or under that path.

Here is an example showing contributions to the Python parser. By default,
contributions will be measured by number of commits:
```
~/repos/cpython$ git who tree Parser/
Parser/.........................Guido van Rossum (182)
├── lexer/......................Pablo Galindo Salgado (5)
│   ├── buffer.c................Lysandros Nikolaou (1)
│   ├── buffer.h................Lysandros Nikolaou (1)
│   ├── lexer.c
│   ├── lexer.h.................Lysandros Nikolaou (1)
│   ├── state.c
│   └── state.h
├── tokenizer/..................Filipe Laíns (1)
│   ├── file_tokenizer.c
│   ├── helpers.c...............Lysandros Nikolaou (1)
│   ├── helpers.h...............Lysandros Nikolaou (1)
│   ├── readline_tokenizer.c....Lysandros Nikolaou (1)
│   ├── string_tokenizer.c......Lysandros Nikolaou (1)
│   ├── tokenizer.h.............Lysandros Nikolaou (1)
│   └── utf8_tokenizer.c........Lysandros Nikolaou (1)
├── Python.asdl.................Benjamin Peterson (14)
├── action_helpers.c............Pablo Galindo Salgado (6)
├── asdl.py.....................Benjamin Peterson (7)
├── asdl_c.py...................Benjamin Peterson (42)
├── myreadline.c
├── parser.c....................Pablo Galindo Salgado (34)
├── peg_api.c...................Lysandros Nikolaou (2)
├── pegen.c.....................Pablo Galindo (33)
├── pegen.h.....................Pablo Galindo Salgado (13)
├── pegen_errors.c..............Pablo Galindo Salgado (16)
├── string_parser.c.............Victor Stinner (10)
├── string_parser.h.............Pablo Galindo Salgado (1)
└── token.c.....................Pablo Galindo Salgado (2)
```

You may notice that some files, like `lexer.c`, are not annotated.
If a file is not annotated, that is because the author who has
most contributed to that file is the same as the author who
has most contributed to the directory containing the file. This is
done to minimize visual noise.

You can force `git-who tree` to annotate every file using the `-a`
flag (for "all"). This flag also prints all file paths that
were discovered while walking the commit history, including those no
longer in the working tree:

```
~/repos/cpython$ git who tree -a Parser/
Parser/.........................Guido van Rossum (182)
├── lexer/......................Pablo Galindo Salgado (5)
│   ├── buffer.c................Lysandros Nikolaou (1)
│   ├── buffer.h................Lysandros Nikolaou (1)
│   ├── lexer.c.................Pablo Galindo Salgado (4)
│   ├── lexer.h.................Lysandros Nikolaou (1)
│   ├── state.c.................Pablo Galindo Salgado (2)
│   └── state.h.................Pablo Galindo Salgado (1)
├── pegen/......................Pablo Galindo (30)
│   ├── parse.c.................Pablo Galindo (16)
│   ├── parse_string.c..........Pablo Galindo (7)
│   ├── parse_string.h..........Pablo Galindo (2)
│   ├── peg_api.c...............Pablo Galindo (3)
│   ├── pegen.c.................Pablo Galindo (17)
│   └── pegen.h.................Pablo Galindo (9)
├── pgen/.......................Pablo Galindo (8)
│   ├── __init__.py.............Pablo Galindo (2)
│   ├── __main__.py.............Pablo Galindo (5)
│   ├── automata.py.............Pablo Galindo (4)
│   ├── grammar.py..............Pablo Galindo (5)
│   ├── keywordgen.py...........Pablo Galindo (3)
│   ├── metaparser.py...........Pablo Galindo (2)
│   ├── pgen.py.................Pablo Galindo (5)
│   └── token.py................Pablo Galindo (4)
├── tokenizer/..................Filipe Laíns (1)
│   ├── file_tokenizer.c........Filipe Laíns (1)
│   ├── helpers.c...............Lysandros Nikolaou (1)
│   ├── helpers.h...............Lysandros Nikolaou (1)
│   ├── readline_tokenizer.c....Lysandros Nikolaou (1)
│   ├── string_tokenizer.c......Lysandros Nikolaou (1)
│   ├── tokenizer.h.............Lysandros Nikolaou (1)
│   └── utf8_tokenizer.c........Lysandros Nikolaou (1)
├── .cvsignore..................Martin v. Löwis (1)
├── Makefile.in.................Guido van Rossum (10)
├── Python.asdl.................Benjamin Peterson (14)
├── acceler.c...................Guido van Rossum (17)
├── action_helpers.c............Pablo Galindo Salgado (6)
├── asdl.py.....................Benjamin Peterson (7)
├── asdl_c.py...................Benjamin Peterson (42)
├── assert.h....................Guido van Rossum (11)
├── bitset.c....................Guido van Rossum (12)
├── firstsets.c.................Guido van Rossum (13)
├── grammar.c...................Guido van Rossum (20)
...
```
(_The above output continues but has been elided for the purposes
of this README._)

Note that, whether or not the `-a` flag is used, commits that
edited files
not in the working tree will still count toward the total displayed
next to ancestor directories of that file. In the above two examples,
Guido van Rossum is shown as the overall highest committer to the
`Parser/` directory, though it takes listing the entire
tree with the `-a` flag to see that most of his commits were to
files that have since been moved or deleted.

Like with the `table` subcommand, you can specify a "commit-ish". This
next example shows changes to the `Parser/` directory that happened
after the 3.10.9 release up to the 3.11.9 release.
```
~/clones/cpython$ git who tree v3.10.9..v3.11.9 -- Parser/
Parser/.................Pablo Galindo Salgado (52)
├── Python.asdl.........Batuhan Taskaya (1)
├── action_helpers.c....Matthieu Dartiailh (1)
├── asdl_c.py...........Batuhan Taskaya (4)
├── myreadline.c........Victor Stinner (1)
├── parser.c
├── pegen.c
├── pegen.h
├── pegen_errors.c......Miss Islington (bot) (8)
└── string_parser.c.....Miss Islington (bot) (4)
```

If a file isn't edited in any of the commits specified by the revision range,
then it won't appear in the output of `git who tree`, even if the file is in
the working tree. This can make `git who tree` useful for visualizing the
changes introduced by a branch.

#### Options
The `tree` subcommand, like the `table` subcommand, supports the `-l`, `-f`,
and `-m` flags. The `-l` flag will annotate each file tree node with the
author who has added or removed the most lines at that path:

```
~/repos/cpython$ git who tree -l Parser/
Parser/.........................Pablo Galindo (72,917 / 47,102)
├── lexer/......................Lysandros Nikolaou (1,668 / 0)
│   ├── buffer.c
│   ├── buffer.h
│   ├── lexer.c
│   ├── lexer.h
│   ├── state.c
│   └── state.h.................Pablo Galindo Salgado (1 / 0)
├── tokenizer/..................Lysandros Nikolaou (1,391 / 0)
│   ├── file_tokenizer.c
│   ├── helpers.c
│   ├── helpers.h
│   ├── readline_tokenizer.c
│   ├── string_tokenizer.c
│   ├── tokenizer.h
│   └── utf8_tokenizer.c
├── Python.asdl.................Benjamin Peterson (120 / 122)
├── action_helpers.c
├── asdl.py.....................Eli Bendersky (276 / 331)
├── asdl_c.py...................Victor Stinner (634 / 496)
├── myreadline.c................Guido van Rossum (365 / 226)
├── parser.c
├── peg_api.c...................Victor Stinner (5 / 46)
├── pegen.c
├── pegen.h
├── pegen_errors.c
├── string_parser.c
├── string_parser.h
└── token.c.....................Serhiy Storchaka (233 / 0)
```

The `-f` flag will pick authors based on files touched and the `-m` flag will
pick an author based on last modification time.

You can limit the depth of the tree printed by using the `-d` flag. The depth
is measured from the current working directory.

The `-a` flag has already been mentioned.

Run `git who tree --help` to see all options available for the `tree` subcommand.

### The `hist` Subcommand
The `hist` subcommand prints out a little bar chart / timeline of commit activity
showing the history of contributions to the repository.

```
~/clones/cpython$ git who hist
1990 ┤ #                                     Guido van Rossum (105)
1991 ┤ ##                                    Guido van Rossum (445)
1992 ┤ ###                                   Guido van Rossum (606)
1993 ┤ #-                                    Guido van Rossum (200)
1994 ┤ ###                                   Guido van Rossum (525)
1995 ┤ ####-                                 Guido van Rossum (869)
1996 ┤ ####---                               Guido van Rossum (961)
1997 ┤ #######--                             Guido van Rossum (1,626)
1998 ┤ #####------                           Guido van Rossum (1,205)
1999 ┤ ###-----                              Fred Drake (755)
2000 ┤ ####------------                      Fred Drake (973)
2001 ┤ #####-----------------                Fred Drake (1,196)
2002 ┤ ###--------------                     Guido van Rossum (543)
2003 ┤ ##------------                        Raymond Hettinger (479)
2004 ┤ ##--------                            Raymond Hettinger (460)
2005 ┤ #----                                 Raymond Hettinger (171)
2006 ┤ ###-------------                      Neal Norwitz (636)
2007 ┤ ####------------                      Guido van Rossum (792)
2008 ┤ ####--------------------              Georg Brandl (1,005)
2009 ┤ #####-----------------------          Benjamin Peterson (1,107)
2010 ┤ #####-------------------------------  Georg Brandl (1,088)
2011 ┤ ####-----------------                 Victor Stinner (877)
2012 ┤ ##------------------                  Antoine Pitrou (466)
2013 ┤ ###--------------                     Victor Stinner (570)
2014 ┤ ###----------                         Victor Stinner (594)
2015 ┤ ###---------                          Victor Stinner (529)
2016 ┤ ##-----------                         Victor Stinner (497)
2017 ┤ ##--------                            Victor Stinner (404)
2018 ┤ ##--------                            Victor Stinner (306)
2019 ┤ ##----------                          Victor Stinner (467)
2020 ┤ ###---------                          Victor Stinner (524)
2021 ┤ ##----------                          Victor Stinner (260)
2022 ┤ ##-------------                       Victor Stinner (366)
2023 ┤ ###---------------                    Victor Stinner (556)
2024 ┤ ##-----------------                   Serhiy Storchaka (321)
2025 ┤ #                                     Bénédikt Tran (27)
```

(Git was only released in 2005, so clearly there has been some version control
metadata imported from another tool!)

The timeline shows the author who made the most commits in each year. The bar
in the bar chart shows their contributions as a proportion of the total
contributions made in that year. (The `#` symbol shows the proportion
of total commits by the "winning" author for that year.)

Like with the other subcommands, you can filter the commits examined to just
those editing files under a given path:

```
~/repos/cpython$ git who hist iOS/
Feb 2024 ┤ #                                     Russell Keith-Magee (1)
Mar 2024 ┤ ####                                  Russell Keith-Magee (4)
Apr 2024 ┤ #-                                    Xie Yanbo (1)
May 2024 ┤
Jun 2024 ┤
Jul 2024 ┤ #                                     Russell Keith-Magee (1)
Aug 2024 ┤ ##                                    Russell Keith-Magee (2)
Sep 2024 ┤ #                                     Russell Keith-Magee (1)
Oct 2024 ┤
Nov 2024 ┤ #                                     Russell Keith-Magee (1)
Dec 2024 ┤ ###-                                  Russell Keith-Magee (3)
Jan 2025 ┤
```
The printed timeline will begin with the date of the first commit modifying
that path.

You can also filter using a commit-ish. This shows the timeline of contributions
since Python's 3.12 release.
```
~/clones/cpython$ git who hist v3.12.0..
May 2023 ┤ ###---------                          Victor Stinner (28)
Jun 2023 ┤ #######--------------------           Victor Stinner (90)
Jul 2023 ┤ ######----------------------------    Victor Stinner (78)
Aug 2023 ┤ #######-------------------------      Victor Stinner (91)
Sep 2023 ┤ ############----------------------    Victor Stinner (157)
Oct 2023 ┤ #####---------------------------      Victor Stinner (68)
Nov 2023 ┤ ###---------------------              Serhiy Storchaka (40)
Dec 2023 ┤ ###-----------------------            Alex Waygood (32)
Jan 2024 ┤ ####-----------------------------     Serhiy Storchaka (43)
Feb 2024 ┤ ####------------------------------    Serhiy Storchaka (42)
Mar 2024 ┤ #####---------------------------      Victor Stinner (59)
Apr 2024 ┤ ###---------------------------        Serhiy Storchaka (37)
May 2024 ┤ ##----------------------------------  Serhiy Storchaka (26)
Jun 2024 ┤ ####------------------------          Victor Stinner (48)
Jul 2024 ┤ ###------------------------           Sam Gross (32)
Aug 2024 ┤ ##-------------------                 Mark Shannon (24)
Sep 2024 ┤ ##---------------------------         Serhiy Storchaka (23)
Oct 2024 ┤ ###----------------------------       Victor Stinner (39)
Nov 2024 ┤ ##-----------------------             Serhiy Storchaka (27)
Dec 2024 ┤ ##------------------                  Bénédikt Tran (18)
Jan 2025 ┤ ##---------                           Bénédikt Tran (26)
```

#### Options
The `hist` subcommand supports the `-l` and `-f` flags but not the `-m` flag:

```
~/repos/cpython$ git who hist -l iOS/
Feb 2024 ┤ ###############                       Russell Keith-Magee (406 / 0)
Mar 2024 ┤ ####################################  Russell Keith-Magee (994 / 32)
Apr 2024 ┤ #                                     Xie Yanbo (2 / 2)
May 2024 ┤
Jun 2024 ┤
Jul 2024 ┤ #                                     Russell Keith-Magee (1 / 1)
Aug 2024 ┤ #                                     Russell Keith-Magee (2 / 0)
Sep 2024 ┤ #                                     Russell Keith-Magee (6 / 0)
Oct 2024 ┤
Nov 2024 ┤ #####                                 Russell Keith-Magee (104 / 28)
Dec 2024 ┤ ##################-                   Russell Keith-Magee (444 / 52)
Jan 2025 ┤
```

Run `git who hist --help` for a full listing of the options supported by the
`hist` subcommand.

### Additional Options for Filtering Commits
All of the `git who` subcommands take these additional options that further
filter the commits that get counted.

The `--author` and `--nauthor` options allow you to specify authors to include
or exclude. Both options can be specified multiple times to include or exclude
multiple authors.

The `--since` option allows you to filter out commits before a certain date.
The option takes a string that gets passed to `git log` to be interpreted. `git
log` can handle some surprising inputs. See git-commit(1) for an explanation of
what is possible.

The following example shows the paths edited by Guido van Rossum over the last
eight months:
```
~/repos/cpython$ git who tree -d 1 --since "nine months ago" --author "Guido van Rossum"
./..................Guido van Rossum (11)
├── .github/........Guido van Rossum (2)
├── Doc/............Guido van Rossum (3)
├── Include/........Guido van Rossum (3)
├── Lib/............Guido van Rossum (1)
├── Modules/........Guido van Rossum (1)
├── Objects/........Guido van Rossum (1)
├── PCbuild/........Guido van Rossum (2)
├── Programs/.......Guido van Rossum (1)
├── Python/.........Guido van Rossum (4)
├── Tools/..........Guido van Rossum (1)
├── configure
└── configure.ac
```

## Git Alias
You can invoke `git-who` as `git who` by setting up an alias in your global Git
config:

```
[alias]
    who = "!git-who"
```

See [here](https://git-scm.com/book/en/v2/Git-Basics-Git-Aliases) for more
information about Git aliases.

## What Exactly Do These Numbers Mean?
### Metrics
The number of **commits** shown for each author is the number of unique commits
found while walking the commit log. When supplying a path argument to `git
who`, the commits walked include only commits modifying the given path(s).
Here, the rules described under the HISTORY SIMPLIFICATION section of Git log
apply—branches in the commit history that do not modify the given path(s) are
pruned away.

The number of **files** shown for each author is the number of unique files
modified in commits by that author. If a file is renamed, it will count twice.

The number of **lines added** and **lines removed** shown for each author is
the number of lines added and removed to files under the supplied path(s) or to
all files in the case of no path arguments. In Git, modifying a line counts as
removing it and then adding the new version of the line.

### Merge Commits
Merge commits are not counted toward any of these metrics. The rationale here
is that merge commits represent a kind of overhead involved in managing the
commit graph and that all novel changes will already have been introduced to
the commit graph by the merge commit's ancestor commits.

You can supply the `--merges` flag to `git who` to change this behavior. The
`--merges` flag forces `git who` to count merge commits toward the commit total
for each author. Merge commits are still ignored for the purposes of the file
total or lines total.

### Differences From `git blame`
Whereas `git blame` starts from the code that exists in the working tree and
identifies the commit that introduced each line, `git who` instead walks some
subset of the commit log tallying contributions. This means that `git blame`
and `git who`, in addition to operating on different levels (individual files
vs file trees), tell you slightly different things.

This is best illustrated through an example. If Bob has made dozens of commits
editing a file, but Alice recently formatted the file and made one big commit
with her style changes, `git blame` will attribute most of the lines in the
file to Alice. `git who`, on the other hand, will rank Bob as the primary
author, at least when sorting by number of commits. In this case, `git who`
seems better suited to answering the question, "Who came up with the code in
this file?"

If instead, Bob made the same commits but Alice came along later and completely
refactored the file, again in one big commit, `git blame` will correctly
attribute most of the lines in the file to her, while `git who` will still list
Bob as the primary author. In this case, `git blame` seems to do a better job
of answering, "Who came up with the code in this file?". That said, the various
subcommands and options of `git who` can give you the full picture of what has
happened here. `git who hist` in particular will show you that Bob was the
primary author until Alice took over.

Ultimately, neither tool quite answers what we want to know, which is "Who came
up with the code in this file?", perhaps because the question is too ambiguous.
`git blame` answers, "Who last modified each line of code in this file?" and
`git who` answers, "Who made the most modifications to this file / this file
tree?"

## DEVELOPMENT
### Test Repository Submodule
Some of the automated tests for `git-who` need to run against a Git repository.
A test repository is attached to this repository as a submodule.

If you want to run the automated tests, you will first need to set up the
submodule:

```
$ git submodule update --init
```

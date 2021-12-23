# Contributing guidelines

Thank you for your interest in contributing to `dusk-blockchain`! In this file,
you will find some guidelines to adhere to when opening issues, pull requests,
and committing code, as well as writing code itself. Following these guidelines
ensures a smooth review process, and increases likelihood of your code getting
merged.

### Table of Contents

* [Coding conventions](#coding-conventions)
    + [Performance](#performance)
    + [Concurrency](#concurrency)
    + [Committing](#committing)
    + [Testing](#testing)
    + [Logging](#logging)
* [Opening issues](#opening-issues)
* [Opening pull requests](#opening-pull-requests)

## Coding conventions

* Ensure to document all exported functions, methods, and structs, according to
  the [Golang documentation guidelines](https://blog.golang.org/godoc).
* Avoid use of gotos as they lead to spaghetti code.
* No else in the if statement, should be the default. Try to always limit the
  amount of indentation used, as it might make it harder to read code.
* New code always needs to be tested. Pull requests will generally not be merged
  if there is no proper test coverage of newly added code.
* If you fix a bug which was hard to find, create/add to the `gotchas.md` file
  in the package. This helps other developers catch the bug much quicker if it
  pops up again.
* If you are adding new code, please include proper documentation outside of
  code comments where it makes sense. Ideally, for adding new features, it helps
  to at the very minimum add a `README.md` which explains purpose, inner
  workings, etc. In some cases, it may also be helpful to include UML diagrams
  which depict the new feature/component (sequence diagrams, component diagrams
  and workflows are often good choices).
* Use the `context` package to manage goroutine lifetimes. For good examples,
  look at how the `Chain` and `Mempool` are managed using contexts.
* Where possible, try to fail and return errors before sending out requests (be
  it GRPC, channels or any other method). Try as much as possible to verify
  input before sending it out.

### Performance

* Wherever possible, when allocating slices, try to allocate the entire slice in
  one go, and then assigning to the indices (or using `copy` where applicable),
  instead of creating an empty slice and then calling `append`. This prevents an
  unnecessary amount of dynamic memory allocations.
* When creating a channel which sole purpose is to send signals, and not data,
  use a `chan struct{}`. The empty struct in Go takes up 0 bytes of space, and
  is thus the most efficient way of sending signals.
* When clearing a map, use a `range` loop, and delete every key. This is more
  efficient than replacing the map with a new instance, as the GC does not have
  to clean this up later.

### Concurrency

* Unless absolutely necessary, try to avoid the use of concurrency. Since Go is
  a language which makes it easy for you to set up concurrent systems, it is
  also easy to go a bit too far and create designs that are far too complex for
  what they are actually trying to solve. Since `dusk-blockchain` is a big,
  complex codebase, simplicity really is key to keeping everything manageable.
  Examples of when you want to use certain synchronization primitives:
    * Channels - when a component is waiting on receiving results/signals from
      different components, which need to do their processing separately to the
      component they report to. In this case, the component that gets reported
      to usually needs to do some work in the background itself. A single-thread
      scenario is not viable here.
    * Atomic - almost never, unless there is a very demonstrable performance
      improvement in a very hot code path, and you can test it enough to prove
      it completely safe.
    * Mutexes - almost always. Single-thread design is often really preferred
      over concurrency, since concurrent systems tend to have a much higher
      amount of potential bugs.
* In addition to these standardly provided primitives, the codebase provides an
  Event Bus and a Request-Response bus (RPCBus) to facilitate communication
  between components that are sectioned off from each other according to
  responsibility.

### Committing

* Rebase your commits generously. Avoid having to open pull requests with a
  massive amount of commits, which all essentially work towards the same issue.
* If possible, try to include all of the code, tests and documentation for one
  issue, into one commit.
* The commit title should be in imperative form. Examples:
    * Implement ...
    * Add ...
    * Change ...
* The commit title should only state what you did. This helps keep commit titles
  short and easy to find.
* The commit description should explain why you did it. Most times, referring to
  the issue itself is enough. Please do so with keywords (`fixes`, `resolves`
  etc.) in order to automatically link issues to commits. This helps to retrace
  commits and it simplifies opening pull requests.
* Do not elaborate on how you did what you did - the code should be
  self-explanatory in this case. If the code is not self-explanatory enough (due
  to performance shortcuts or the sort), please explain this in comments, rather
  than in the commit.
* Make the lines in your commit max 80 characters. This ensures that everything
  is properly readable on GitHub.

### Testing

* One test should test one thing. Test code is still code, and so we should try
  to separate concerns and make it as readable as possible. This rule may be
  broken in cases where it does not make sense to separate concerns.
* If there are internal and external tests related to the package, then mark the
  files with name\_in\_ test.go and name\_ex\_test.go
* Make use of the `testify` package where possible. This clearly separates test
  code from production code, since the error handling is noticably different. It
  also provides useful functionality for checking equality, making sure certain
  fields are empty/not empty etc.
* If there are multiple assertions, make sure to instantiate an assert object,
  like such:

```golang
func TestSomething(t *testing.T) {
assert := assert.New(t)
// ...
}
```

* Where appropriate, make use
  of [table testing](https://www.tmap.net/wiki/decision-table-test-dtt). An
  example can be found in the `reduction` package tests.
* If a file of unit tests needs some preliminary setup (for instance, database
  entries), make use of the [TestMain](https://golang.org/pkg/testing/#hdr-Main)
  function. However, in most cases, unit tests can function perfectly fine on
  their own.

### Logging

* Use `github.com/sirupsen/logrus` for logging rather than `fmt`
* As a general rule use:
    * `log.Traceln` \(lowest level debug info\) mostly without dynamic
      parameters to debug flow \(use rarely and only when testing\)
    * `log.Debugln` \(lower level debug info\) mostly with dynamic parameters to
      debug process a process \(use sparingly\)
    * `log.Infoln` \(useful info for integration\) mostly with dynamic
      parameters to for general logging \(use sparingly\)
    * `log.Warnln` \(situations potentially leading to errors or lower priority
      error detection\) always with dynamic parameters \(use at will\)
    * `log.Errorln` \(recoverable errors\) always with dynamic parameters
      \(almost never used\)
    * `log.Fatalln` \(always before panic\) always with dynamic parameters
      \(almost never used\)
* Set dynamic parameters in the fields, not in the message. For instance set
  parameters in the following way:

```text
log.WithFields(log.Fields{ "param1" : param1, "param2": param2 }).Infoln("Some message")
```

Do not do the following:

```text
log.Infoln("Some message with params1 %v and param2 %v", param1, param2)
```

In order to allow for the monitoring system to filter out the logs properly,
ensure you properly name all relevant fields, and always include a field which
informs the monitoring of which process the log entry is coming from.

## Opening issues

* An issue should be as atomic as possible. If you wish to create a tracking
  issue for a broader scope, please tag it with the `Epic` label.
* Always write issues in imperative form - start with a verb. This makes it
  clear what needs to be done. Some examples:
    * Implement ...
    * Add ...
    * Change ...
* Always try to tag the issue with relevant labels. This helps us sort issues
  more properly, and prioritize accordingly.

## Opening pull requests

* Try to stick to one pull request per issue; in some cases, this rule makes
  sense to be broken, but do so at your own discretion. Oftentimes, things can
  be tackled individually, and this simplifies the review process massively. A
  good way to know if you did it right, is when the pull request automatically
  comes up with a proper title, description, and properly links to the issue(s)
  it fixes - you'll have one commit, for one issue, in one PR.
* Much like issues, try to:
    * Have an imperative title
    * Explain why this code was written in the description
    * Use keywords, linking to issues that are being resolved (`fixes`
      , `resolves` etc.)
* Tag the pull request with relevant labels. In many cases, it makes sense to
  use the same labels as found in the issue(s) it fixes.
* Try to open a draft pull request as soon as you can - this helps others to
  track your progress, and makes it easier for people to help out if needed.
* Your pull request is ready to be merged, once it has one approving review, and
  all GitHub Actions have been successful. As a rule of thumb, the person who
  opens the PR, is the one who merges it, so you are free to merge it at your
  discretion after that.
* Once you have merged your pull request, you should delete your branch, in
  order to keep the branch list clean.

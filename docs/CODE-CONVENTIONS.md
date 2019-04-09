## Code Conventions

### Testing

- One test should test one thing. Test code, is still code and so we should try to separate concerns and make it as readable as possible.

- If there are internal and external tests related to the package, then mark the files with name*in* test.go and name_ex_test.go

- Test methods should Ideally be made for every file, even if they are compositions of others. For example, in bulletproof, the vector file has it’s own test files, it is used in many different files in BP which also contain their own respective test files.

### General

- Reflection should be a last ditch effort, usually when a package uses reflection in a strongly typed language, it is a sign of bad architecture. Sometimes you will need a big switch statement.

- Avoid empty interface, same reason as reflection.

- Exposed methods, functions and Structs are to be commented by starting with the name of the object. <- Found this in a few of my unfinished packages.

- Pushing to repo should be prefaced with the package/folder that is being modified.

- Performance comes second to readability in most cases, because if it cannot be read, it most likely cannot be maintained, and or improved upon.

- Try to be simple, if we don’t need to use a smart way to do something, then it would be best not to.

- Use branches when modifying code, try to avoid pushing to main, unless it is a small correction.

- Avoid use of gotos as they lead to spaghetti code.

- Try to make nested if statements maximum two levels deep.

- No else in the if statement, should be the default.

#### Logging

- Use `github.com/sirupse/logrus` for logging rather than `fmt`
- As a general rule use:
  - `log.Traceln` (lowest level debug info) mostly without dynamic parameters to debug flow (use rarely and only when testing)
  - `log.Debugln` (lower level debug info) mostly with dynamic parameters to debug process a process (use sparingly)
  - `log.Infoln` (useful info for integration) mostly with dynamic parameters to for general logging (use sparingly)
  - `log.Warnln` (situations potentially leading to errors or lower priority error detection) always with dynamic parameters (use at will)
  - `log.Errorln` (recoverable errors) always with dynamic parameters (almost never used)
  - `log.Fatalln` (always before panic) always with dynamic parameters (almost never used)
- Set dynamic parameters in the fields, not in the message. For instance set parameters in the following way:

```
log.WithFields(log.Fields{ "param1" : param1, "param2": param2 }).Infoln("Some message")
```

do not do the following:

```
log.Infoln("Some message with params1 %v and param2 %v", param1, param2)
```

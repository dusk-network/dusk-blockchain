## Code Conventions

### Testing 

- One test should test one thing. Test code, is still code and so we should try to separate concerns and make it as readable as possible.

- If there are internal and external tests related to the package, then mark the files with name_in_ test.go and name_ex_test.go 

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
# Remarks

The `remarks` branch is used to add comments, ask questions, and propose improvements to the code. 
We call each of these comments, questions, and proposals, a `remark`.

A `remark` is added as a comment **ABOVE** the line/block of code to which it refers to, and it starts with `"REMARK:"`.  
For instance:
```
// REMARK: This is a 'remark'
```
or
```
/* REMARK: This is another 'remark' */
```

<!-- TODO: link PR when ready -->
Remarks committed to the `remarks` branch can be discussed in the _Remarks_ PR.
This will be a special "ongoing" PR, with each commit including one `remark`, or multiple ones if they are related.

## Discussion and Implementation

Ideally, a remark will have the following lifecycle:
1. The remark is committed and pushed to `remarks`.
3. In the PR, the remark is discussed by means of `comments` (as in a normal PR).
4. If the remark does not need to be addressed, it is deleted with a new commit.
5. If the remark needs to be addressed, 
    a new Issue is generate starting from one of the comments.
6. The new Issue is managed as usual (with its own PR).
7. When the Issue gets closed (i.e., its PR has been merged),  
    the `remarks` branch is rebased to `main`, deleting the resolved remark.


# Code Review

The `code-review` branch is used to add comments, ask questions, and propose improvements to the code. 
We call each of these comments, questions, and proposals, a `remark`.

A `remark` is added as a _comment_ **ABOVE** the line/block of code to which it refers to, and it starts with `"REMARK:"`.  
For instance:
```
/* REMARK: This is a 'remark' */
```

<!-- TODO: link PR when ready -->
Remarks committed to `code-review` can be discussed in the _Code Review_ PR.
This is a special "ongoing" PR, with each commit including one `remark`, or multiple ones if they are related.

## Discussion and Implementation

Ideally, a remark will have the following lifecycle:
1. The remark is committed and pushed to `code-review`.
3. In the _Code Review_ PR, the remark is discussed by means of `comments` (as in a normal PR).
4. If the remark does not need to be addressed, it is deleted with a new commit.
5. If the remark needs to be addressed, 
    a new Issue is created starting from one of the comments.
6. The new Issue is managed as usual (with its own PR).
7. When the Issue gets closed (i.e., its PR is merged),  
    the `code-review` branch is rebased to `main`, deleting the resolved remark.


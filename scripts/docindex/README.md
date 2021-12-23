# [scripts/docindex](./scripts/docindex)

With all packages and directories in the repository having a consistently
structured header and contents page position, it is simple to then walk the
directory tree and concatenate this section with generated rubrics that
hyperlink to the package folder where the readme with the header is found.

<!-- ToC start -->
##  Contents

   1. [Implementation notes](#implementation-notes)
1. [to regenerate this file's table of contents:](#to-regenerate-this-files-table-of-contents:)
<!-- ToC end -->

## Implementation notes

- create file walker closure which
    - matches on README.md
    - cuts off the ToC marker and below
    - regex match `[#].*\n\n` to elide heading line and space, return
      non-matched segment
    - constructs header with appropriate `##` level of the package path and
      relative hyperlink to the package folder, prepends to 'blurb' section from
      remainder of previous
- join all resulting generated into one with the document header located beside
  the docindex generator script source files, and overwrite
  [index.md](../../index.md)

idea: automatically generate taglist from words in the whole document that are
some limited number of the least frequently occurring in english words, to catch
the terms that the blurb may be missing for searching. A kind of poor man's
fulltext index, but very easy to construct.

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)

# [scripts/md2godoc](./scripts/md2godoc)

Automatically create godoc package headers from README.md's

<!-- ToC start -->

## Contents

<!-- ToC end -->

## Abstract 

Go has a reasonably decent inline documentation generation protocol called
'godoc', which captures and processes text found in specific locations in a
source file comment blocks.

Due to the ambiguous definition of where the package header text should come
from, and the many quirks of the format, basically, the comment on line zero,
and as such many projects, such as [btcd](https://github.com/btcsuite/btcd)
designate a `doc.go` to hold this, and leave the rest of the headers empty.

Well, Dusk sources have a copyright header in this position also. But also,
[godoc for this repository](https://pkg.go.dev/github.com/dusk-network/dusk-blockchain)
- the site - shows the `README.md`.

Here is [an example](https://pkg.go.dev/github.com/dusk-network/dusk-blockchain@v0.4.3/pkg/core/consensus#section-readme)
which shows the readme is displayed.

This is all very well, but [with the docs.go in place](https://pkg.go.dev/github.com/golang/mock/gomock)
the documentation being shown is in the source code comments.

To cut a very long story short, this script lets you treat the markdown as
the single place to edit the documentation, while exporting this with
minimum pain to also show correctly on godoc, even offline.

## Implementation Notes

- [https://github.com/princjef/gomarkdoc](https://github.com/princjef/gomarkdoc)

  This tool will be used to convert the layout to the more simplified markup 
  used by godoc for headers, tables, and the like.

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)

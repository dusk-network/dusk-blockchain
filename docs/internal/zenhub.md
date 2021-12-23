# ZenHub workflow guidelines

*NOTE*: this document is for internal usage (within Dusk Network) only, and
describes the workflow of an internal tool we use for issue management, ZenHub.

This document should bring you up to speed on how to properly use ZenHub, so
that the board is always in the right shape and your teammates can easily tell
how your work is progressing.

### Table of Contents

* [Pipelines](#pipelines)
    + [New Issues](#new-issues)
    + [Icebox](#icebox)
    + [Backlog](#backlog)
    + [To Do](#to-do)
    + [In Progress](#in-progress)
    + [Review/QA](#review-qa)
* [Workflow](#workflow)
    + [Opening and managing Epics](#opening-and-managing-epics)
    + [Opening and managing issues](#opening-and-managing-issues)
        - [Blockers](#blockers)
        - [Pull requests](#pull-requests)

## Pipelines

This section describes the purpose of all the different pipelines found on
the `Node-UX` board.

### New Issues

This is where issues will land once they are created, and aren't assigned any
pipeline to begin with. Ideally, you shouldn't leave any issues here, as this
pipeline is purely just there to catch new issues.

### Icebox

The icebox is primarily for lower-priority issues which are not part of the
current iteration (1 week worth of work). These issues can be picked up if one
is bored, blocked or generally has nothing better to work on.

### Backlog

Contains the issues that can be worked on at any time, but are not part of the
current iteration. Tickets coming from `New Issues` generally end up here.

### To Do

Issues in this category are part of the current iteration, and are ready to be
picked up. Usually contains higher priority issues, or issues associated with an
Epic which is in progress.

### In Progress

When an issue is actively being worked on, it belongs here. As a rule of thumb,
try to only have a singular issue in progress at all times. If issues are
related to one another, it would of course make sense to put them both into In
Progress. A good way to tell if this is appropriate, is if they will be fixed in
the same PR. If this is the case, the developer should establish dependencies
between relevant issues, by selecting on the different tickets which issues are
blocking which.

### Review/QA

Once you have finished work on your issue, it should be moved into this
category, so that your teammates will know a review is needed.

## Workflow

In order to keep the board in good order, it is fundamental that anyone working
on it takes care to actively maintain it and follow a set of rules. These rules
will be outlined below.

### Opening and managing Epics

An Epic is a tracking issue which encompasses any number of smaller, more atomic
issues, which work towards a broader goal. In the title, it often talks about a
certain feature or change which needs to be made, which will take a longer time
to complete and needs to be split up into individual pieces of work, by way of
smaller issues. In the Epic, it is important to give a good bit of information
about what is trying to be achieved, why it is necessary, and how it will be
done, roughly.

Once the Epic is opened, it's important to list all related issues under it, so
that it is easy to track the progress of this Epic. This can either be done from
the issues themselves (the Epic category on the bottom right), or from the Epic,
by clicking `Add issues to this Epic` at the bottom of the page.

Ensure to only tag it with the `Epic` issue label. To open an Epic, click the +
at the top right of the page, and select `Epic`.

### Opening and managing issues

Besides following the guidelines in
the [contribution document](../../CONTRIBUTING.md), there are a couple of extra
things you need to take care of when managing issues through ZenHub.

When opening an issue, ensure that you:

* Assign the relevant person, if possible
* Add all the relevant labels
* Group it under the relevant epic, if applicable
* Immediately move it to the proper pipeline

As you work on stuff, make sure you move issues around on the board as you
switch tasks. For overall clarity, it is essential that you actively maintain an
up-to-date view of the board at all times.

#### Blockers

If an issue is blocked by another one, you can easily mark this on ZenHub by
clicking on a ticket, and then clicking on `add dependency`. Here, you can
either select issues that the current one is blocked by, or the issues that the
current one is blocking. Doing this helps to give a more clear overview of how
different issues relate to one another, and it helps the rest of the team to
prioritize work.

#### Pull requests

Once you open a pull request for an issue, draft or otherwise, it is important
that you *link* it to the issue it fixes. This is done by opening the PR ticket,
scrolling down, and clicking `Connect Issue`, and then selecting the relevant
issue. If it fixes multiple issues, link it to the one with the lowest priority

- this ensures that work is done on the more pressing issues first. Ensure the
  pull request has the same set of labels as the issue, and assign yourself to
  the PR. Having PRs and issues linked, ensures the board is less scattered, and
  makes it easier for yourself to make sure all tickets are in the right place.

#!/bin/sh

# Send a notification to the Dusk Build Status Channel once a travis CI build completes.
# The b
#

BOT_URL="https://api.telegram.org/bot${TELEGRAM_TOKEN}/sendMessage"

PARSE_MODE="Markdown"

if [ $TRAVIS_TEST_RESULT -ne 0 ]; then
    build_status="failed"
else
    build_status="succeeded"
fi

send_msg () {
    curl -s -X POST ${BOT_URL} -d chat_id=$TELEGRAM_CHAT_ID \
        -d text="$1" -d parse_mode=${PARSE_MODE}
}

send_msg "
-------------------------------------
Travis build *${build_status}!*
\`Repository:  ${TRAVIS_REPO_SLUG}\`
\`Branch:      ${TRAVIS_BRANCH}\`
*Commit Msg:*
${TRAVIS_COMMIT_MESSAGE}
[Job Log here](${TRAVIS_JOB_WEB_URL})
--------------------------------------
"


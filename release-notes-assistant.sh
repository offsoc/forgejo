#!/bin/bash
# Copyright twenty-panda <twenty-panda@posteo.com>
# SPDX-License-Identifier: MIT

label_worth=worth
label_bug=bug
label_feature=feature
label_ui=forgejo/ui
label_breaking=breaking
label_localization=internationalization

payload=$(mktemp)
pr=$(mktemp)
trap "rm $payload $pr" EXIT

function test_main() {
  set -ex
  PS4='${BASH_SOURCE[0]}:$LINENO: ${FUNCNAME[0]}:  '

  test_payload_labels $label_worth $label_breaking $label_feature
  test "$(categorize)" = 'AA Breaking features'

  test_payload_labels $label_worth $label_breaking $label_bug
  test "$(categorize)" = 'AB Breaking bug fixes'

  test_payload_labels $label_worth $label_breaking
  test "$(categorize)" = 'ZC Breaking changes without a feature or bug label'

  test_payload_labels $label_worth $label_ui $label_feature
  test "$(categorize)" = 'BA User Interface features'

  test_payload_labels $label_worth $label_ui $label_bug
  test "$(categorize)" = 'BB User Interface bug fixes'

  test_payload_labels $label_worth $label_ui
  test "$(categorize)" = 'ZD User Interface changes without a feature or bug label'

  test_payload_labels $label_worth $label_feature
  test "$(categorize)" = 'CA Features'

  test_payload_labels $label_worth $label_bug
  test "$(categorize)" = 'CB Bug fixes'

  test_payload_labels $label_worth $label_localization
  test "$(categorize)" = 'DA Localization'

  test_payload_labels $label_worth
  test "$(categorize)" = 'ZE Other changes without a feature or bug label'

  test_payload_labels
  test "$(categorize)" = 'ZF Included for completness but not worth a release note'

  test_payload_draft "feat!: breaking feature"
  test "$(categorize)" = 'AA Breaking features'

  test_payload_draft "fix!: breaking bug fix"
  test "$(categorize)" = 'AB Breaking bug fixes'

  test_payload_draft "feat: feature"
  test "$(categorize)" = 'CA Features'

  test_payload_draft "fix: bug fix"
  test "$(categorize)" = 'CB Bug fixes'

  test_payload_draft "something with no prefix"
  test "$(categorize)" = 'ZE Other changes without a feature or bug label'
}

function main() {
  cat >$payload
  categorize
}

function categorize() {
  #
  # If this is a backport, refer to the original PR to figure
  # out the classification.
  #
  if $(jq --raw-output .IsBackportedFrom <$payload); then
    jq --raw-output '.BackportedFrom[0]' <$payload >$pr
  else
    jq --raw-output '.Pr' <$payload >$pr
  fi

  labels=$(jq --raw-output '.labels[].name' <$pr)

  #
  # Was this PR labeled `worth a release note`?
  #
  if echo "$labels" | grep --quiet $label_worth; then
    worth=true
  else
    worth=false
  fi

  #
  # If there was no release-notes/N.md file and it is not
  # worth a release note, just forget about it.
  #
  if test -z "$(jq --raw-output .Draft <$payload)"; then
    if ! $worth; then
      echo -n ZF Included for completness but not worth a release note
      exit 0
    fi
  fi

  is_ui=false
  is_bug=false
  is_feature=false
  is_localization=false
  is_breaking=false

  #
  # first try to figure out the category from the labels
  #
  case "$labels" in
  *$label_bug*)
    is_bug=true
    ;;
  *$label_feature*)
    is_feature=true
    ;;
  *$label_localization*)
    is_localization=true
    ;;
  esac

  case "$labels" in
  *$label_breaking*)
    is_breaking=true
    ;;
  esac

  case "$labels" in
  *$label_ui*)
    is_ui=true
    ;;
  esac

  #
  # then try the prefix of the release note
  #
  if ! $is_bug && ! $is_feature; then
    draft="$(jq --raw-output .Draft <$payload)"
    case "$draft" in
    fix!:*)
      is_bug=true
      is_breaking=true
      ;;
    fix:*)
      is_bug=true
      ;;
    feat!:*)
      is_feature=true
      is_breaking=true
      ;;
    feat:*)
      is_feature=true
      ;;
    esac
  fi

  if $is_bug; then
    if $(jq --raw-output .IsBackportedTo <$payload); then
      #
      # if it has been backported, it was in the release notes of an older stable release
      # and does not need to be in this more recent release notes
      #
      echo -n ZG Already announced in the release notes of an older stable release
      exit 0
    fi
  fi

  if $is_breaking; then
    if $is_feature; then
      echo -n AA Breaking features
    elif $is_bug; then
      echo AB Breaking bug fixes
    else
      echo -n ZC Breaking changes without a feature or bug label
    fi
  elif $is_ui; then
    if $is_feature; then
      echo -n BA User Interface features
    elif $is_bug; then
      echo -n BB User Interface bug fixes
    else
      echo -n ZD User Interface changes without a feature or bug label
    fi
  elif $is_localization; then
    echo -n DA Localization
  else
    if $is_feature; then
      echo -n CA Features
    elif $is_bug; then
      echo -n CB Bug fixes
    else
      echo -n ZE Other changes without a feature or bug label
    fi
  fi
}

function test_payload_labels() {
  local label1="$1"
  local label2="$2"
  local label3="$3"
  local label4="$4"

  cat >$payload <<EOF
{
  "Pr": {
    "labels": [
      {
        "name": "$label1"
      },
      {
        "name": "$label2"
      },
      {
        "name": "$label3"
      },
      {
        "name": "$label4"
      }
    ]
  },
  "IsBackportedFrom": false,
  "Draft": ""
}
EOF
}

function test_payload_draft() {
  local draft="$1"

  cat >$payload <<EOF
{
  "Pr": {
    "labels": [
      {
        "name": "$label_worth"
      }
    ]
  },
  "IsBackportedFrom": false,
  "Draft": "$draft"
}
EOF
}

"${@:-main}"
